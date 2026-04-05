package main

import (
	"crypto/rand"
	"database/sql"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	_ "modernc.org/sqlite"
)

var db *sql.DB

var clientsMutex sync.Mutex
var clients = make(map[*websocket.Conn]bool)

// Klassiske Matador Huslejer: [0 huse, 1 hus, 2 huse, 3 huse, 4 huse, Hotel]
// Rederier og Bryggerier har vi sat til faste standard-satser, indtil vi evt. får terninger.
var RentTable = map[string][]int{
	"Rødovrevej":         {50, 250, 750, 2250, 4000, 6000},
	"Hvidovrevej":        {50, 250, 750, 2250, 4000, 6000},
	"Roskildevej":        {100, 600, 1800, 5400, 8000, 11000},
	"Valby Langgade":     {100, 600, 1800, 5400, 8000, 11000},
	"Allégade":           {150, 800, 2000, 6000, 9000, 12000},
	"Frederiksberg Allé": {200, 1000, 3000, 9000, 12500, 15000},
	"Bülowsvej":          {200, 1000, 3000, 9000, 12500, 15000},
	"Gammel Kongevej":    {250, 1250, 3750, 10000, 14000, 18000},
	"Bernstorffsvej":     {300, 1400, 4000, 11000, 15000, 19000},
	"Hellerupvej":        {300, 1400, 4000, 11000, 15000, 19000},
	"Strandvejen":        {350, 1600, 4400, 12000, 16000, 20000},
	"Trianglen":          {350, 1800, 5000, 14000, 17500, 21000},
	"Østerbrogade":       {350, 1800, 5000, 14000, 17500, 21000},
	"Grønningen":         {400, 2000, 6000, 15000, 18500, 22000},
	"Bredgade":           {450, 2200, 6600, 16000, 19500, 23000},
	"Kongens Nytorv":     {450, 2200, 6600, 16000, 19500, 23000},
	"Østergade":          {500, 2400, 7200, 17000, 20500, 24000},
	"Amagertorv":         {550, 2600, 7800, 18000, 22000, 25000},
	"Vimmelskaftet":      {550, 2600, 7800, 18000, 22000, 25000},
	"Nygade":             {600, 3000, 9000, 20000, 24000, 28000},
	"Frederiksberggade":  {700, 3500, 10000, 22000, 26000, 30000},
	"Rådhuspladsen":      {1000, 4000, 12000, 28000, 34000, 40000},
	"D.F.D.S.":           {500, 500, 500, 500, 500, 500},
	"Ø.K.":               {500, 500, 500, 500, 500, 500},
	"Bornholm":           {500, 500, 500, 500, 500, 500},
	"Mols-Linien":        {500, 500, 500, 500, 500, 500},
	"Tuborg":             {400, 400, 400, 400, 400, 400},
	"Carlsberg":          {400, 400, 400, 400, 400, 400},
}

type WSMessage struct {
	Action string                 `json:"action"`
	Data   map[string]interface{} `json:"data"`
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func generatePIN() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(9000))
	return fmt.Sprintf("%04d", n.Int64()+1000)
}

func generateID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}

func broadcast(message map[string]interface{}) {
	clientsMutex.Lock()
	defer clientsMutex.Unlock()
	for client := range clients {
		err := client.WriteJSON(message)
		if err != nil {
			client.Close()
			delete(clients, client)
		}
	}
}

func seedProperties(gameID string) {
	props := map[string]int{
		"Rødovrevej": 1200, "Hvidovrevej": 1200, "Roskildevej": 2000, "Valby Langgade": 2000, "Allégade": 2400,
		"Frederiksberg Allé": 2800, "Bülowsvej": 2800, "Gammel Kongevej": 3200, "Bernstorffsvej": 3600, "Hellerupvej": 3600,
		"Strandvejen": 4000, "Trianglen": 4400, "Østerbrogade": 4400, "Grønningen": 4800, "Bredgade": 5200,
		"Kongens Nytorv": 5200, "Østergade": 5600, "Amagertorv": 6000, "Vimmelskaftet": 6000, "Nygade": 6400,
		"Frederiksberggade": 7000, "Rådhuspladsen": 8000, "D.F.D.S.": 4000, "Ø.K.": 4000, "Bornholm": 4000,
		"Mols-Linien": 4000, "Tuborg": 3000, "Carlsberg": 3000,
	}
	for name, price := range props {
		db.Exec("INSERT INTO properties (id, game_id, property_name, price, owner_id, houses, is_mortgaged) VALUES (?, ?, ?, ?, ?, ?, ?)", generateID(), gameID, name, price, "", 0, false)
	}
}

func getProperties(gameID string) []map[string]interface{} {
	rows, _ := db.Query("SELECT property_name, price, owner_id, houses, is_mortgaged FROM properties WHERE game_id = ?", gameID)
	defer rows.Close()
	var properties []map[string]interface{}
	for rows.Next() {
		var name, owner string
		var price, houses int
		var isMortgaged bool
		rows.Scan(&name, &price, &owner, &houses, &isMortgaged)
		properties = append(properties, map[string]interface{}{"name": name, "price": price, "owner_id": owner, "houses": houses, "is_mortgaged": isMortgaged})
	}
	return properties
}

func getPlayers(gameID string) []map[string]interface{} {
	rows, _ := db.Query("SELECT id, name, balance, in_jail FROM players WHERE game_id = ?", gameID)
	defer rows.Close()
	var players []map[string]interface{}
	for rows.Next() {
		var id, name string
		var balance int
		var inJail bool
		rows.Scan(&id, &name, &balance, &inJail)
		players = append(players, map[string]interface{}{"id": id, "name": name, "balance": balance, "in_jail": inJail})
	}
	return players
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	clientsMutex.Lock()
	clients[conn] = true
	clientsMutex.Unlock()
	defer func() { clientsMutex.Lock(); delete(clients, conn); clientsMutex.Unlock(); conn.Close() }()

	for {
		var msg WSMessage
		err := conn.ReadJSON(&msg)
		if err != nil {
			break
		}

		switch msg.Action {

		case "create_game":
			gameID := generateID()
			pin := generatePIN()
			db.Exec("INSERT INTO games (id, pin_code, status) VALUES (?, ?, ?)", gameID, pin, "waiting")
			seedProperties(gameID)
			conn.WriteJSON(map[string]interface{}{"event": "game_created", "pin": pin, "game_id": gameID})

		case "join_game":
			pin, _ := msg.Data["pin"].(string)
			playerName, _ := msg.Data["name"].(string)
			var gameID string
			if err := db.QueryRow("SELECT id FROM games WHERE pin_code = ? AND status = 'waiting'", pin).Scan(&gameID); err != nil {
				conn.WriteJSON(map[string]interface{}{"event": "error", "message": "Ugyldig PIN-kode."})
				continue
			}
			playerID := generateID()
			db.Exec("INSERT INTO players (id, game_id, name, balance, in_jail) VALUES (?, ?, ?, ?, ?)", playerID, gameID, playerName, 30000, false)
			conn.WriteJSON(map[string]interface{}{"event": "join_success", "player_id": playerID, "game_id": gameID, "name": playerName})
			broadcast(map[string]interface{}{"event": "player_joined", "game_id": gameID, "player_id": playerID, "player_name": playerName})

		case "start_game":
			gameID, _ := msg.Data["game_id"].(string)
			db.Exec("UPDATE games SET status = 'active' WHERE id = ?", gameID)
			broadcast(map[string]interface{}{"event": "game_started", "game_id": gameID, "players": getPlayers(gameID), "properties": getProperties(gameID)})

		case "give_pass_start_bonus":
			playerID, _ := msg.Data["player_id"].(string)
			gameID, _ := msg.Data["game_id"].(string)
			db.Exec("UPDATE players SET balance = balance + 4000 WHERE id = ?", playerID)
			broadcast(map[string]interface{}{"event": "players_updated", "game_id": gameID, "players": getPlayers(gameID)})
			conn.WriteJSON(map[string]interface{}{"event": "admin_success", "message": "4.000 kr. udbetalt!"})

		case "send_to_jail":
			playerID, _ := msg.Data["player_id"].(string)
			gameID, _ := msg.Data["game_id"].(string)
			db.Exec("UPDATE players SET in_jail = TRUE WHERE id = ?", playerID)
			broadcast(map[string]interface{}{"event": "players_updated", "game_id": gameID, "players": getPlayers(gameID)})
			conn.WriteJSON(map[string]interface{}{"event": "admin_success", "message": "Spiller smidt i fængsel!"})

		case "pay_bail":
			playerID, _ := msg.Data["player_id"].(string)
			gameID, _ := msg.Data["game_id"].(string)
			var currentBalance int
			db.QueryRow("SELECT balance FROM players WHERE id = ?", playerID).Scan(&currentBalance)
			if currentBalance < 1000 {
				conn.WriteJSON(map[string]interface{}{"event": "error", "message": "Du har ikke 1.000 kr. til kaution!"})
				continue
			}
			db.Exec("UPDATE players SET balance = balance - 1000, in_jail = FALSE WHERE id = ?", playerID)
			broadcast(map[string]interface{}{"event": "players_updated", "game_id": gameID, "players": getPlayers(gameID)})
			conn.WriteJSON(map[string]interface{}{"event": "admin_success", "message": "Kaution betalt! Du er fri."})

		// --- NYT: BYG HUSE ---
		case "build_house":
			gameID, _ := msg.Data["game_id"].(string)
			propName, _ := msg.Data["property_name"].(string)
			playerID, _ := msg.Data["player_id"].(string)
			var houses, currentBalance int
			db.QueryRow("SELECT houses FROM properties WHERE game_id = ? AND property_name = ?", gameID, propName).Scan(&houses)
			db.QueryRow("SELECT balance FROM players WHERE id = ?", playerID).Scan(&currentBalance)

			if houses >= 5 {
				conn.WriteJSON(map[string]interface{}{"event": "error", "message": "Der er allerede et hotel her!"})
				continue
			}
			if currentBalance < 1000 {
				conn.WriteJSON(map[string]interface{}{"event": "error", "message": "Du har ikke 1.000 kr. til at bygge!"})
				continue
			}

			db.Exec("UPDATE players SET balance = balance - 1000 WHERE id = ?", playerID)
			db.Exec("UPDATE properties SET houses = houses + 1 WHERE game_id = ? AND property_name = ?", gameID, propName)

			broadcast(map[string]interface{}{"event": "properties_updated", "game_id": gameID, "properties": getProperties(gameID)})
			broadcast(map[string]interface{}{"event": "players_updated", "game_id": gameID, "players": getPlayers(gameID)})

		// --- NYT: HUSLEJE ---
		case "demand_rent":
			gameID, _ := msg.Data["game_id"].(string)
			propName, _ := msg.Data["property_name"].(string)

			// Slå op i databasen: Hvor mange huse står der lige nu?
			var houses int
			err := db.QueryRow("SELECT houses FROM properties WHERE game_id = ? AND property_name = ?", gameID, propName).Scan(&houses)

			if err != nil {
				continue
			}

			// Sikkerheds-tjek: Man kan max have 5 huse (hotel) i arrayet
			if houses > 5 {
				houses = 5
			}

			// Slå den faste pris op i vores Matador-Tabel!
			amount := RentTable[propName][houses]

			// Send den præcise regning afsted
			broadcast(map[string]interface{}{
				"event":         "rent_demanded",
				"game_id":       gameID,
				"target_id":     msg.Data["target_id"],
				"owner_id":      msg.Data["owner_id"],
				"owner_name":    msg.Data["owner_name"],
				"property_name": propName,
				"amount":        amount,
			})

		case "pay_rent":
			gameID, _ := msg.Data["game_id"].(string)
			fromID, _ := msg.Data["from_id"].(string)
			toID, _ := msg.Data["to_id"].(string)
			amountFloat, _ := msg.Data["amount"].(float64)
			amount := int(amountFloat)

			var currentBalance int
			if err := db.QueryRow("SELECT balance FROM players WHERE id = ?", fromID).Scan(&currentBalance); err != nil || currentBalance < amount {
				conn.WriteJSON(map[string]interface{}{"event": "error", "message": "Du har ikke penge nok! Pantsæt noget først."})
				continue
			}

			db.Exec("UPDATE players SET balance = balance - ? WHERE id = ?", amount, fromID)
			db.Exec("UPDATE players SET balance = balance + ? WHERE id = ?", amount, toID)

			broadcast(map[string]interface{}{"event": "players_updated", "game_id": gameID, "players": getPlayers(gameID)})
			conn.WriteJSON(map[string]interface{}{"event": "admin_success", "message": "Husleje betalt!"}) // Sender grøn notifikation til betaleren

		case "mortgage_property":
			gameID, _ := msg.Data["game_id"].(string)
			propName, _ := msg.Data["property_name"].(string)
			playerID, _ := msg.Data["player_id"].(string)
			var price int
			db.QueryRow("SELECT price FROM properties WHERE game_id = ? AND property_name = ?", gameID, propName).Scan(&price)
			payout := price / 2
			db.Exec("UPDATE properties SET is_mortgaged = TRUE WHERE game_id = ? AND property_name = ?", gameID, propName)
			db.Exec("UPDATE players SET balance = balance + ? WHERE id = ?", payout, playerID)
			broadcast(map[string]interface{}{"event": "properties_updated", "game_id": gameID, "properties": getProperties(gameID)})
			broadcast(map[string]interface{}{"event": "players_updated", "game_id": gameID, "players": getPlayers(gameID)})

		case "unmortgage_property":
			gameID, _ := msg.Data["game_id"].(string)
			propName, _ := msg.Data["property_name"].(string)
			playerID, _ := msg.Data["player_id"].(string)
			var price, currentBalance int
			db.QueryRow("SELECT price FROM properties WHERE game_id = ? AND property_name = ?", gameID, propName).Scan(&price)
			db.QueryRow("SELECT balance FROM players WHERE id = ?", playerID).Scan(&currentBalance)
			cost := int(float64(price) * 0.55)

			if currentBalance < cost {
				conn.WriteJSON(map[string]interface{}{"event": "error", "message": "Du har ikke råd!"})
				continue
			}
			db.Exec("UPDATE properties SET is_mortgaged = FALSE WHERE game_id = ? AND property_name = ?", gameID, propName)
			db.Exec("UPDATE players SET balance = balance - ? WHERE id = ?", cost, playerID)
			broadcast(map[string]interface{}{"event": "properties_updated", "game_id": gameID, "properties": getProperties(gameID)})
			broadcast(map[string]interface{}{"event": "players_updated", "game_id": gameID, "players": getPlayers(gameID)})

		case "transfer_money":
			gameID, _ := msg.Data["game_id"].(string)
			fromPlayerID, _ := msg.Data["from_player_id"].(string)
			toPlayerID, _ := msg.Data["to_player_id"].(string)
			amountFloat, _ := msg.Data["amount"].(float64)
			amount := int(amountFloat)
			if amount <= 0 {
				continue
			}
			var currentBalance int
			if err := db.QueryRow("SELECT balance FROM players WHERE id = ?", fromPlayerID).Scan(&currentBalance); err != nil || currentBalance < amount {
				conn.WriteJSON(map[string]interface{}{"event": "error", "message": "Afvist: Du har ikke nok penge på kontoen!"})
				continue
			}
			db.Exec("UPDATE players SET balance = balance - ? WHERE id = ?", amount, fromPlayerID)
			if toPlayerID != "bank" {
				db.Exec("UPDATE players SET balance = balance + ? WHERE id = ?", amount, toPlayerID)
			}
			broadcast(map[string]interface{}{"event": "players_updated", "game_id": gameID, "players": getPlayers(gameID)})

		case "offer_property":
			broadcast(map[string]interface{}{
				"event": "property_offered", "game_id": msg.Data["game_id"], "buyer_id": msg.Data["buyer_id"],
				"seller_id": msg.Data["seller_id"], "seller_name": msg.Data["seller_name"],
				"property_name": msg.Data["property_name"], "price": msg.Data["price"],
			})

		case "accept_property":
			gameID, _ := msg.Data["game_id"].(string)
			buyerID, _ := msg.Data["buyer_id"].(string)
			sellerID, _ := msg.Data["seller_id"].(string)
			propName, _ := msg.Data["property_name"].(string)
			priceFloat, _ := msg.Data["price"].(float64)
			price := int(priceFloat)

			var currentBalance int
			if err := db.QueryRow("SELECT balance FROM players WHERE id = ?", buyerID).Scan(&currentBalance); err != nil || currentBalance < price {
				conn.WriteJSON(map[string]interface{}{"event": "error", "message": "Afvist: Du har ikke råd!"})
				continue
			}

			db.Exec("UPDATE players SET balance = balance - ? WHERE id = ?", price, buyerID)
			if sellerID != "bank" && sellerID != "" {
				db.Exec("UPDATE players SET balance = balance + ? WHERE id = ?", price, sellerID)
			}
			db.Exec("UPDATE properties SET owner_id = ? WHERE game_id = ? AND property_name = ?", buyerID, gameID, propName)

			broadcast(map[string]interface{}{"event": "property_bought_success", "game_id": gameID, "buyer_id": buyerID, "property_name": propName})
			broadcast(map[string]interface{}{"event": "players_updated", "game_id": gameID, "players": getPlayers(gameID)})
			broadcast(map[string]interface{}{"event": "properties_updated", "game_id": gameID, "properties": getProperties(gameID)})

		case "reconnect":
			gameID, _ := msg.Data["game_id"].(string)
			playerID, _ := msg.Data["player_id"].(string)
			isHost, _ := msg.Data["is_host"].(bool)
			var status, pin string
			if err := db.QueryRow("SELECT status, pin_code FROM games WHERE id = ?", gameID).Scan(&status, &pin); err != nil {
				conn.WriteJSON(map[string]interface{}{"event": "reconnect_failed"})
				continue
			}

			players := getPlayers(gameID)
			var myName string
			var myBalance int
			var inJail bool
			for _, p := range players {
				if p["id"] == playerID {
					myName = p["name"].(string)
					myBalance = p["balance"].(int)
					inJail = p["in_jail"].(bool)
				}
			}
			if !isHost && myName == "" {
				conn.WriteJSON(map[string]interface{}{"event": "reconnect_failed"})
				continue
			}

			conn.WriteJSON(map[string]interface{}{
				"event": "reconnect_success", "game_id": gameID, "status": status, "pin": pin,
				"is_host": isHost, "player_id": playerID, "name": myName, "balance": myBalance, "in_jail": inJail,
				"players": players, "properties": getProperties(gameID),
			})
		}
	}
}

func main() {
	var err error
	db, err = sql.Open("sqlite", "./matador.db?_pragma=journal_mode(WAL)")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	createTablesSQL := `
	CREATE TABLE IF NOT EXISTS games (id TEXT PRIMARY KEY, pin_code TEXT, status TEXT, created_at DATETIME DEFAULT CURRENT_TIMESTAMP);
	CREATE TABLE IF NOT EXISTS players (id TEXT PRIMARY KEY, game_id TEXT, name TEXT, balance INTEGER, in_jail BOOLEAN DEFAULT FALSE, get_out_of_jail_cards INTEGER DEFAULT 0);
	CREATE TABLE IF NOT EXISTS properties (id TEXT PRIMARY KEY, game_id TEXT, property_name TEXT, price INTEGER, owner_id TEXT, houses INTEGER DEFAULT 0, is_mortgaged BOOLEAN DEFAULT FALSE);`
	db.Exec(createTablesSQL)

	http.HandleFunc("/ws", handleWebSocket)
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)

	log.Println("Matador-serveren kører på http://localhost:8080")
	http.ListenAndServe(":8080", nil)
}
