// Project 2: Number Guessing Game
//
// Computer picks a random number 1-100; you guess, with hints, until you get
// it or run out of attempts. Then it asks if you want another round.
package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
)

const (
	minNumber   = 1
	maxNumber   = 100
	maxAttempts = 7
)

func readLine(reader *bufio.Reader, prompt string) string {
	fmt.Print(prompt)
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(line)
}

// playRound runs one full game and reports whether the player won.
func playRound(reader *bufio.Reader) (won bool) {
	target := rand.Intn(maxNumber-minNumber+1) + minNumber
	fmt.Printf("\nI'm thinking of a number between %d and %d. You have %d guesses.\n",
		minNumber, maxNumber, maxAttempts)

	// Using a "while-style" for loop (condition only, no init/post) instead of
	// the classic three-part form: that lets `continue` skip incrementing
	// `attempt` on invalid input, so a bad guess doesn't cost the player a try.
	attempt := 1
guessing:
	for attempt <= maxAttempts {
		input := readLine(reader, fmt.Sprintf("Guess %d/%d: ", attempt, maxAttempts))

		guess, err := strconv.Atoi(input)
		if err != nil {
			fmt.Println("Please enter a whole number.")
			continue guessing // re-prompt without incrementing attempt
		}

		switch {
		case guess < minNumber || guess > maxNumber:
			fmt.Printf("Stay within %d-%d.\n", minNumber, maxNumber)
			continue guessing
		case guess == target:
			fmt.Printf("Correct! It was %d. You got it in %d guesses.\n", target, attempt)
			won = true
			break guessing
		case guess < target:
			fmt.Println("Too low.")
		case guess > target:
			fmt.Println("Too high.")
		}

		if attempt == maxAttempts && guess != target {
			fmt.Printf("Out of guesses — the number was %d.\n", target)
		}
		attempt++
	}
	return won
}

func main() {
	reader := bufio.NewReader(os.Stdin)
	wins, rounds := 0, 0

rounds:
	for {
		rounds++
		if playRound(reader) {
			wins++
		}

		again := readLine(reader, "\nPlay again? (y/n): ")
		switch strings.ToLower(again) {
		case "y", "yes":
			continue rounds
		default:
			break rounds
		}
	}

	fmt.Printf("\nFinal score: %d win(s) out of %d round(s). Thanks for playing!\n", wins, rounds)
}
