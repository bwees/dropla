package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"app/internal/database"
	"app/internal/repositories"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/term"
)

// handleCLI runs a subcommand and reports whether it consumed the invocation.
//
// reset-password exists for the locked-out sole admin, who has no other route
// back in: there is no email recovery anywhere in dropla. It opens the database
// directly rather than talking to a running server, so it works either way.
func handleCLI() bool {
	if len(os.Args) < 2 || os.Args[1] != "reset-password" {
		return false
	}

	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: dropla reset-password <username>")
		fmt.Fprintln(os.Stderr, "the new password is read from stdin, never from the command line")
		os.Exit(2)
	}

	username := os.Args[2]

	password, err := readPassword()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	if len(password) < 8 {
		fmt.Fprintln(os.Stderr, "error: password must be at least 8 characters")
		os.Exit(1)
	}

	db := database.NewDatabase()
	users := repositories.NewUsersRepository(db)
	sessions := repositories.NewSessionsRepository(db)

	user, err := users.GetByUsername(username)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	if user == nil {
		fmt.Fprintf(os.Stderr, "error: no such user %q\n", username)
		os.Exit(1)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	if err := users.UpdatePassword(user.ID, string(hash)); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	// A reset drops every session, same as a voluntary change.
	if err := sessions.DeleteForUser(user.ID); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	fmt.Printf("password reset for %s; all sessions signed out\n", username)
	return true
}

// readPassword prompts on a terminal and reads a piped value otherwise, so
// `echo 'secret' | docker exec -i dropla dropla reset-password alice` works.
// A password in argv would land in shell history and in ps output.
func readPassword() (string, error) {
	if term.IsTerminal(int(os.Stdin.Fd())) {
		fmt.Fprint(os.Stderr, "New password: ")
		b, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(os.Stderr)
		return string(b), err
	}

	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	return strings.TrimRight(line, "\r\n"), err
}
