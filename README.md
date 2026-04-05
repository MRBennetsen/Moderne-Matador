# 🏦 Moderne Matador

**Moderne Matador** er en digital "Banking & Property Management" companion-app til det klassiske brætspil Matador. Slut med krøllede papirpenge og skøder, der bliver væk – alt kører lynhurtigt i realtid over WebSockets.

Dette projekt er bygget som en moderne web-app med fokus på lynhurtig performance, session-styring og en autentisk spiloplevelse.

## ✨ Features

- **Realtids-økonomi:** Alle transaktioner sker øjeblikkeligt via WebSockets. Ingen behov for refresh!
- **Digital Ejendomsmægler:** Køb og sælg grunde med et "MobilePay-agtigt" godkendelses-flow og konfetti-effekter 🎉.
- **Automatisk Leje-beregning:** Serveren har de originale Matador-huslejer hardcoded. Slår du op i tabellen? Nej, serveren gør det for dig baseret på antal huse!
- **Huse & Hoteller:** Byg dit imperium (🏠🏠🏠🏠🏨) direkte fra din telefon.
- **Pantsætning:** Kommer du i knibe? Pantsæt dine grunde for 50% af værdien og køb dem fri senere (mod 10% rente).
- **Fængselssystem:** Admin kan smide spillere i spjældet 🚔. Spilleren skal betale 1.000 kr. i kaution for at slippe ud.
- **Session Persistence:** Siden genindlæst midt i det hele? Intet problem. Systemet husker dit spil og din saldo via LocalStorage og automatisk reconnect-logik.
- **Bank-kontrol:** En dedikeret Admin-skærm til banken for at styre passér-start bonusser og fængselsstraf.

## 🛠 Teknisk Stack

- **Backend:** [Go (Golang)](https://golang.org/) – Valgt for sin ekstreme hurtighed og effektive håndtering af samtidige forbindelser.
- **Database:** [SQLite](https://sqlite.org/) (via modernc.org/sqlite) – En "Pure Go" implementation, der gør det muligt at køre projektet uden eksterne C-dependencies.
- **Kommunikation:** [WebSockets (Gorilla)](https://github.com/gorilla/websocket) – Sikrer tovejs-kommunikation i realtid.
- **Frontend:** Vanilla JS, HTML5 og CSS3. Ingen tunge frameworks – bare ren og hurtig kode.
