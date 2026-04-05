package main

import (
	"crypto/rand"
	"database/sql"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"sort"
	"sync"

	"github.com/gorilla/websocket"
	_ "modernc.org/sqlite"
)

var db *sql.DB

var clientsMutex sync.Mutex
var clients = make(map[*websocket.Conn]bool)

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

var RentTable = map[string][]int{
	"Rødovrevej": {50, 250, 750, 2250, 4000, 6000}, "Hvidovrevej": {50, 250, 750, 2250, 4000, 6000},
	"Roskildevej": {100, 600, 1800, 5400, 8000, 11000}, "Valby Langgade": {100, 600, 1800, 5400, 8000, 11000},
	"Allégade": {150, 800, 2000, 6000, 9000, 12000}, "Frederiksberg Allé": {200, 1000, 3000, 9000, 12500, 15000},
	"Bülowsvej": {200, 1000, 3000, 9000, 12500, 15000}, "Gammel Kongevej": {250, 1250, 3750, 10000, 14000, 18000},
	"Bernstorffsvej": {300, 1400, 4000, 11000, 15000, 19000}, "Hellerupvej": {300, 1400, 4000, 11000, 15000, 19000},
	"Strandvejen": {350, 1600, 4400, 12000, 16000, 20000}, "Trianglen": {350, 1800, 5000, 14000, 17500, 21000},
	"Østerbrogade": {350, 1800, 5000, 14000, 17500, 21000}, "Grønningen": {400, 2000, 6000, 15000, 18500, 22000},
	"Bredgade": {450, 2200, 6600, 16000, 19500, 23000}, "Kongens Nytorv": {450, 2200, 6600, 16000, 19500, 23000},
	"Østergade": {500, 2400, 7200, 17000, 20500, 24000}, "Amagertorv": {550, 2600, 7800, 18000, 22000, 25000},
	"Vimmelskaftet": {550, 2600, 7800, 18000, 22000, 25000}, "Nygade": {600, 3000, 9000, 20000, 24000, 28000},
	"Frederiksberggade": {700, 3500, 10000, 22000, 26000, 30000}, "Rådhuspladsen": {1000, 4000, 12000, 28000, 34000, 40000},
	"D.F.D.S.": {500, 500, 500, 500, 500, 500}, "Ø.K.": {500, 500, 500, 500, 500, 500},
	"Bornholm": {500, 500, 500, 500, 500, 500}, "Mols-Linien": {500, 500, 500, 500, 500, 500},
	"Tuborg": {400, 400, 400, 400, 400, 400}, "Carlsberg": {400, 400, 400, 400, 400, 400},
}

type ChanceCard struct {
	Text   string
	Action string
	Value  int
}

var ChanceCardsDeck = []ChanceCard{
	{"Dit yndlings memecoin er gået til månen! 🚀 Modtag 1.000 kr.", "add_money", 1000},
	{"Din MobilePay Box fra weekendens bytur er gjort op. Modtag 500 kr.", "add_money", 500},
	{"Du har fået penge tilbage i SKAT! 💸 Ryk frem til START og modtag 4.000 kr.", "add_money", 4000},
	{"Du har solgt dit aflagte tøj på Vinted. Modtag 800 kr.", "add_money", 800},
	{"Du fandt en glemt Bitcoin på en gammel harddisk! ₿ Modtag 3.000 kr.", "add_money", 3000},
	{"Dit TikTok-cover gik viralt! Modtag 1.500 kr.", "add_money", 1500},
	{"Du afmeldte dit PureGym-abonnement. Modtag 500 kr.", "add_money", 500},
	{"Varmechecken er gået ind! ☀️ Modtag 1.000 kr.", "add_money", 1000},
	{"Glemt p-skive i din Tesla. Betal 200 kr.", "sub_money", 200},
	{"Din elbil skal på værksted. Betal 2.000 kr.", "sub_money", 2000},
	{"Blitzet på et tunet el-løbehjul! 🛴 Betal 1.000 kr.", "sub_money", 1000},
	{"Glemt at afmelde LinkedIn Premium. Betal 500 kr.", "sub_money", 500},
	{"For meget iskaffe med havremælk. ☕ Betal 800 kr.", "sub_money", 800},
	{"Data-roaming uden for EU. 📱 Betal 1.200 kr.", "sub_money", 1200},
	{"Snydt af falsk MitID-SMS. Betal 2.000 kr.", "sub_money", 2000},
	{"Købt Taylor Swift billetter af billethaj. 🎫 Betal 3.000 kr.", "sub_money", 3000},
	{"Taget i at dele Netflix-kodeord! 🚔 Gå direkte i fængsel.", "go_to_jail", 0},
	{"Brugt ChatGPT til eksamen! 🤖 Gå direkte i fængsel.", "go_to_jail", 0},
	{"Ny CO2-afgift! 🏭 Betal 800 kr. pr. hus og 2.300 kr. pr. hotel.", "property_tax", 0},
	{"Vurderingsstyrelsen har lavet rod! 🏚️ Efterbetal 800 kr. pr. hus og 2.300 kr. pr. hotel.", "property_tax", 0},
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
	rows, _ := db.Query("SELECT id, name, balance, in_jail, is_bankrupt, get_out_of_jail_cards FROM players WHERE game_id = ?", gameID)
	defer rows.Close()
	var players []map[string]interface{}
	for rows.Next() {
		var id, name string
		var balance, jailCards int
		var inJail, isBankrupt bool
		rows.Scan(&id, &name, &balance, &inJail, &isBankrupt, &jailCards)
		players = append(players, map[string]interface{}{"id": id, "name": name, "balance": balance, "in_jail": inJail, "is_bankrupt": isBankrupt, "jail_cards": jailCards})
	}
	return players
}

func getPlayerName(id string) string {
	if id == "bank" || id == "" {
		return "Banken"
	}
	var name string
	db.QueryRow("SELECT name FROM players WHERE id = ?", id).Scan(&name)
	return name
}

func broadcastLog(gameID, message string) {
	db.Exec("INSERT INTO logs (game_id, message) VALUES (?, ?)", gameID, message)
	broadcast(map[string]interface{}{"event": "new_log", "game_id": gameID, "message": message})
}

func getLogs(gameID string) []string {
	rows, _ := db.Query("SELECT message FROM (SELECT id, message FROM logs WHERE game_id = ? ORDER BY id DESC LIMIT 25) ORDER BY id ASC", gameID)
	defer rows.Close()
	var logs []string
	for rows.Next() {
		var msg string
		rows.Scan(&msg)
		logs = append(logs, msg)
	}
	return logs
}

func broadcastPool(gameID string) {
	var pool int
	db.QueryRow("SELECT parking_pool FROM games WHERE id = ?", gameID).Scan(&pool)
	broadcast(map[string]interface{}{"event": "pool_updated", "game_id": gameID, "pool": pool})
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

		case "ping":
			continue

		case "get_history":
			rows, _ := db.Query("SELECT substr(created_at, 1, 16), winner_name FROM games WHERE status = 'finished' ORDER BY created_at DESC LIMIT 5")
			var history []map[string]string
			for rows.Next() {
				var date, winner string
				rows.Scan(&date, &winner)
				history = append(history, map[string]string{"date": date, "winner": winner})
			}
			rows.Close()
			conn.WriteJSON(map[string]interface{}{"event": "history_data", "history": history})

		case "create_game":
			gameID := generateID()
			pin := generatePIN()
			db.Exec("INSERT INTO games (id, pin_code, status, parking_pool, winner_name) VALUES (?, ?, ?, ?, ?)", gameID, pin, "waiting", 0, "")
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
			db.Exec("INSERT INTO players (id, game_id, name, balance, in_jail, is_bankrupt, get_out_of_jail_cards) VALUES (?, ?, ?, ?, ?, ?, ?)", playerID, gameID, playerName, 30000, false, false, 0)
			conn.WriteJSON(map[string]interface{}{"event": "join_success", "player_id": playerID, "game_id": gameID, "name": playerName})
			broadcast(map[string]interface{}{"event": "player_joined", "game_id": gameID, "player_id": playerID, "player_name": playerName})

		case "start_game":
			gameID, _ := msg.Data["game_id"].(string)
			db.Exec("UPDATE games SET status = 'active' WHERE id = ?", gameID)
			broadcastLog(gameID, "🏁 Spillet er startet!")
			broadcast(map[string]interface{}{"event": "game_started", "game_id": gameID, "players": getPlayers(gameID), "properties": getProperties(gameID), "logs": getLogs(gameID)})
			broadcastPool(gameID)

		case "end_game":
			gameID, _ := msg.Data["game_id"].(string)
			rows, _ := db.Query("SELECT id, name, balance, is_bankrupt FROM players WHERE game_id = ?", gameID)

			type PlayerResult struct {
				Name     string `json:"name"`
				NetWorth int    `json:"net_worth"`
			}
			var results []PlayerResult

			for rows.Next() {
				var pID, pName string
				var bal int
				var isBankrupt bool
				rows.Scan(&pID, &pName, &bal, &isBankrupt)

				if isBankrupt {
					results = append(results, PlayerResult{Name: pName, NetWorth: 0})
					continue
				}

				netWorth := bal
				pRows, _ := db.Query("SELECT price, is_mortgaged, houses FROM properties WHERE game_id = ? AND owner_id = ?", gameID, pID)
				for pRows.Next() {
					var price, houses int
					var mortgaged bool
					pRows.Scan(&price, &mortgaged, &houses)
					if mortgaged {
						netWorth += price / 2
					} else {
						netWorth += price
					}
					netWorth += houses * 1000 // 1000 kr pr hus
				}
				pRows.Close()
				results = append(results, PlayerResult{Name: pName, NetWorth: netWorth})
			}
			rows.Close()

			sort.Slice(results, func(i, j int) bool { return results[i].NetWorth > results[j].NetWorth })

			winner := "Ingen"
			if len(results) > 0 {
				winner = results[0].Name
			}

			db.Exec("UPDATE games SET status = 'finished', winner_name = ? WHERE id = ?", winner, gameID)
			broadcastLog(gameID, fmt.Sprintf("🛑 Spillet blev afsluttet af Banken! %s vandt med en formue på %d kr.", winner, results[0].NetWorth))
			broadcast(map[string]interface{}{"event": "game_ended", "game_id": gameID, "leaderboard": results, "winner": winner})

		case "give_pass_start_bonus":
			playerID, _ := msg.Data["player_id"].(string)
			gameID, _ := msg.Data["game_id"].(string)
			db.Exec("UPDATE players SET balance = balance + 4000 WHERE id = ?", playerID)
			broadcastLog(gameID, fmt.Sprintf("🎁 %s passerede START (+4.000 kr.)", getPlayerName(playerID)))
			broadcast(map[string]interface{}{"event": "players_updated", "game_id": gameID, "players": getPlayers(gameID)})
			conn.WriteJSON(map[string]interface{}{"event": "admin_success", "message": "4.000 kr. udbetalt!"})

		case "send_to_jail":
			playerID, _ := msg.Data["player_id"].(string)
			gameID, _ := msg.Data["game_id"].(string)
			db.Exec("UPDATE players SET in_jail = TRUE WHERE id = ?", playerID)
			broadcastLog(gameID, fmt.Sprintf("🚔 %s er blevet smidt i fængsel!", getPlayerName(playerID)))
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
			db.Exec("UPDATE games SET parking_pool = parking_pool + 1000 WHERE id = ?", gameID)

			broadcastLog(gameID, fmt.Sprintf("🔓 %s har betalt kaution (1000 kr. i Bødekassen)", getPlayerName(playerID)))
			broadcast(map[string]interface{}{"event": "players_updated", "game_id": gameID, "players": getPlayers(gameID)})
			broadcastPool(gameID)
			conn.WriteJSON(map[string]interface{}{"event": "admin_success", "message": "Kaution betalt! Du er fri."})

		case "give_jail_card":
			playerID, _ := msg.Data["player_id"].(string)
			gameID, _ := msg.Data["game_id"].(string)
			db.Exec("UPDATE players SET get_out_of_jail_cards = get_out_of_jail_cards + 1 WHERE id = ?", playerID)
			broadcastLog(gameID, fmt.Sprintf("🃏 %s modtog et Løsladelseskort!", getPlayerName(playerID)))
			broadcast(map[string]interface{}{"event": "players_updated", "game_id": gameID, "players": getPlayers(gameID)})
			conn.WriteJSON(map[string]interface{}{"event": "admin_success", "message": "Kort uddelt!"})

		case "use_jail_card":
			playerID, _ := msg.Data["player_id"].(string)
			gameID, _ := msg.Data["game_id"].(string)
			db.Exec("UPDATE players SET in_jail = FALSE, get_out_of_jail_cards = get_out_of_jail_cards - 1 WHERE id = ?", playerID)
			broadcastLog(gameID, fmt.Sprintf("🔓 %s brugte et Løsladelseskort!", getPlayerName(playerID)))
			broadcast(map[string]interface{}{"event": "players_updated", "game_id": gameID, "players": getPlayers(gameID)})

		case "payout_pool":
			playerID, _ := msg.Data["player_id"].(string)
			gameID, _ := msg.Data["game_id"].(string)
			var pool int
			db.QueryRow("SELECT parking_pool FROM games WHERE id = ?", gameID).Scan(&pool)
			if pool == 0 {
				conn.WriteJSON(map[string]interface{}{"event": "error", "message": "Bødekassen er tom!"})
				continue
			}

			db.Exec("UPDATE players SET balance = balance + ? WHERE id = ?", pool, playerID)
			db.Exec("UPDATE games SET parking_pool = 0 WHERE id = ?", gameID)

			broadcastLog(gameID, fmt.Sprintf("🚗💰 %s landede på Gratis Parkering og vandt %d kr.!", getPlayerName(playerID), pool))
			broadcast(map[string]interface{}{"event": "players_updated", "game_id": gameID, "players": getPlayers(gameID)})
			broadcast(map[string]interface{}{"event": "pool_won", "game_id": gameID, "target_id": playerID, "amount": pool})
			broadcastPool(gameID)

		case "draw_chance_card":
			playerID, _ := msg.Data["player_id"].(string)
			gameID, _ := msg.Data["game_id"].(string)
			n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(ChanceCardsDeck))))
			card := ChanceCardsDeck[n.Int64()]
			var changeAmount int = 0
			isPositive := false

			switch card.Action {
			case "add_money":
				db.Exec("UPDATE players SET balance = balance + ? WHERE id = ?", card.Value, playerID)
				changeAmount = card.Value
				isPositive = true
			case "sub_money":
				db.Exec("UPDATE players SET balance = balance - ? WHERE id = ?", card.Value, playerID)
				db.Exec("UPDATE games SET parking_pool = parking_pool + ? WHERE id = ?", card.Value, gameID)
				changeAmount = -card.Value
			case "go_to_jail":
				db.Exec("UPDATE players SET in_jail = TRUE WHERE id = ?", playerID)
			case "property_tax":
				rows, _ := db.Query("SELECT houses FROM properties WHERE game_id = ? AND owner_id = ?", gameID, playerID)
				totalTax := 0
				for rows.Next() {
					var h int
					rows.Scan(&h)
					if h == 5 {
						totalTax += 2300
					} else if h > 0 {
						totalTax += (h * 800)
					}
				}
				rows.Close()
				db.Exec("UPDATE players SET balance = balance - ? WHERE id = ?", totalTax, playerID)
				db.Exec("UPDATE games SET parking_pool = parking_pool + ? WHERE id = ?", totalTax, gameID)
				changeAmount = -totalTax
				card.Text = fmt.Sprintf("%s\n(Din samlede regning blev: %d kr.)", card.Text, totalTax)
			}
			broadcastLog(gameID, fmt.Sprintf("❓ %s trak et Chancekort!", getPlayerName(playerID)))
			broadcast(map[string]interface{}{"event": "chance_card_drawn", "game_id": gameID, "target_id": playerID, "card_text": card.Text, "is_positive": isPositive, "amount_changed": changeAmount})
			broadcast(map[string]interface{}{"event": "players_updated", "game_id": gameID, "players": getPlayers(gameID)})
			broadcastPool(gameID)

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
			broadcastLog(gameID, fmt.Sprintf("🏗️ %s byggede på %s", getPlayerName(playerID), propName))
			broadcast(map[string]interface{}{"event": "properties_updated", "game_id": gameID, "properties": getProperties(gameID)})
			broadcast(map[string]interface{}{"event": "players_updated", "game_id": gameID, "players": getPlayers(gameID)})

		case "demand_rent":
			gameID, _ := msg.Data["game_id"].(string)
			propName, _ := msg.Data["property_name"].(string)
			var houses int
			if err := db.QueryRow("SELECT houses FROM properties WHERE game_id = ? AND property_name = ?", gameID, propName).Scan(&houses); err != nil {
				continue
			}
			if houses > 5 {
				houses = 5
			}
			amount := RentTable[propName][houses]
			broadcast(map[string]interface{}{"event": "rent_demanded", "game_id": gameID, "target_id": msg.Data["target_id"], "owner_id": msg.Data["owner_id"], "owner_name": msg.Data["owner_name"], "property_name": propName, "amount": amount})

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
			broadcastLog(gameID, fmt.Sprintf("🏠 %s betalte %d kr. i husleje til %s", getPlayerName(fromID), amount, getPlayerName(toID)))
			broadcast(map[string]interface{}{"event": "players_updated", "game_id": gameID, "players": getPlayers(gameID)})
			conn.WriteJSON(map[string]interface{}{"event": "admin_success", "message": "Husleje betalt!"})

		case "mortgage_property":
			gameID, _ := msg.Data["game_id"].(string)
			propName, _ := msg.Data["property_name"].(string)
			playerID, _ := msg.Data["player_id"].(string)
			var price int
			db.QueryRow("SELECT price FROM properties WHERE game_id = ? AND property_name = ?", gameID, propName).Scan(&price)
			payout := price / 2
			db.Exec("UPDATE properties SET is_mortgaged = TRUE WHERE game_id = ? AND property_name = ?", gameID, propName)
			db.Exec("UPDATE players SET balance = balance + ? WHERE id = ?", payout, playerID)
			broadcastLog(gameID, fmt.Sprintf("🛑 %s pantsatte %s", getPlayerName(playerID), propName))
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
			broadcastLog(gameID, fmt.Sprintf("✅ %s købte %s fri", getPlayerName(playerID), propName))
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
			broadcastLog(gameID, fmt.Sprintf("💸 %s overførte %d kr. til %s", getPlayerName(fromPlayerID), amount, getPlayerName(toPlayerID)))
			broadcast(map[string]interface{}{"event": "players_updated", "game_id": gameID, "players": getPlayers(gameID)})

		case "offer_property":
			broadcast(map[string]interface{}{
				"event": "property_offered", "game_id": msg.Data["game_id"], "buyer_id": msg.Data["buyer_id"],
				"seller_id": msg.Data["seller_id"], "seller_name": msg.Data["seller_name"], "property_name": msg.Data["property_name"], "price": msg.Data["price"],
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
			broadcastLog(gameID, fmt.Sprintf("📜 %s købte %s af %s for %d kr.", getPlayerName(buyerID), propName, getPlayerName(sellerID), price))
			broadcast(map[string]interface{}{"event": "property_bought_success", "game_id": gameID, "buyer_id": buyerID, "property_name": propName})
			broadcast(map[string]interface{}{"event": "players_updated", "game_id": gameID, "players": getPlayers(gameID)})
			broadcast(map[string]interface{}{"event": "properties_updated", "game_id": gameID, "properties": getProperties(gameID)})

		case "declare_bankruptcy":
			gameID, _ := msg.Data["game_id"].(string)
			playerID, _ := msg.Data["player_id"].(string)
			targetID, _ := msg.Data["target_id"].(string)
			var balance int
			db.QueryRow("SELECT balance FROM players WHERE id = ?", playerID).Scan(&balance)
			if balance > 0 {
				db.Exec("UPDATE players SET balance = 0 WHERE id = ?", playerID)
				if targetID != "bank" && targetID != "" {
					db.Exec("UPDATE players SET balance = balance + ? WHERE id = ?", balance, targetID)
				}
			}
			if targetID == "bank" || targetID == "" {
				db.Exec("UPDATE properties SET owner_id = '', houses = 0, is_mortgaged = FALSE WHERE game_id = ? AND owner_id = ?", gameID, playerID)
			} else {
				db.Exec("UPDATE properties SET owner_id = ? WHERE game_id = ? AND owner_id = ?", targetID, gameID, playerID)
			}
			db.Exec("UPDATE players SET is_bankrupt = TRUE WHERE id = ?", playerID)
			broadcastLog(gameID, fmt.Sprintf("💀 %s GIK FALLIT og overgav alt til %s", getPlayerName(playerID), getPlayerName(targetID)))
			broadcast(map[string]interface{}{"event": "properties_updated", "game_id": gameID, "properties": getProperties(gameID)})
			broadcast(map[string]interface{}{"event": "players_updated", "game_id": gameID, "players": getPlayers(gameID)})

			// Slettet auto-vinder kode for Fallit (nu har vi jo Slut Spil-knappen!)

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
			var myBalance, jailCards int
			var inJail, isBankrupt bool
			for _, p := range players {
				if p["id"] == playerID {
					myName = p["name"].(string)
					myBalance = p["balance"].(int)
					inJail = p["in_jail"].(bool)
					isBankrupt = p["is_bankrupt"].(bool)
					jailCards = p["jail_cards"].(int)
				}
			}
			if !isHost && myName == "" {
				conn.WriteJSON(map[string]interface{}{"event": "reconnect_failed"})
				continue
			}

			conn.WriteJSON(map[string]interface{}{
				"event": "reconnect_success", "game_id": gameID, "status": status, "pin": pin,
				"is_host": isHost, "player_id": playerID, "name": myName, "balance": myBalance, "in_jail": inJail, "is_bankrupt": isBankrupt, "jail_cards": jailCards,
				"players": players, "properties": getProperties(gameID), "logs": getLogs(gameID),
			})
			broadcastPool(gameID)
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
	CREATE TABLE IF NOT EXISTS games (id TEXT PRIMARY KEY, pin_code TEXT, status TEXT, parking_pool INTEGER DEFAULT 0, winner_name TEXT DEFAULT '', created_at DATETIME DEFAULT CURRENT_TIMESTAMP);
	CREATE TABLE IF NOT EXISTS players (id TEXT PRIMARY KEY, game_id TEXT, name TEXT, balance INTEGER, in_jail BOOLEAN DEFAULT FALSE, get_out_of_jail_cards INTEGER DEFAULT 0, is_bankrupt BOOLEAN DEFAULT FALSE);
	CREATE TABLE IF NOT EXISTS properties (id TEXT PRIMARY KEY, game_id TEXT, property_name TEXT, price INTEGER, owner_id TEXT, houses INTEGER DEFAULT 0, is_mortgaged BOOLEAN DEFAULT FALSE);
	CREATE TABLE IF NOT EXISTS logs (id INTEGER PRIMARY KEY AUTOINCREMENT, game_id TEXT, message TEXT, created_at DATETIME DEFAULT CURRENT_TIMESTAMP);`
	db.Exec(createTablesSQL)

	http.HandleFunc("/ws", handleWebSocket)
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)

	log.Println("Matador-serveren kører på http://localhost:8080")
	http.ListenAndServe(":8080", nil)
}
