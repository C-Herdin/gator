package main

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq"

	"github.com/C-Herdin/Gator/internal/config"
	"github.com/C-Herdin/Gator/internal/database"
)

func main() {
	cfg, err := config.Read()
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
	db, err := sql.Open("postgres", cfg.DbURL)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
	dbQueries := database.New(db)
	var s *state = &state{db: dbQueries, cfg: &cfg}
	var cmds = commands{handlers: make(map[string]func(*state, command) error)}
	cmds.registerAll()
	if len(os.Args) < 2 {
		fmt.Println("not enough arguments provided")
		os.Exit(1)
	}
	if err := cmds.run(s, command{os.Args[1], os.Args[2:]}); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}
