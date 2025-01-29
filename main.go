package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/gdamore/tcell/v2"
	"github.com/go-vgo/robotgo"
	"github.com/rivo/tview"
)

var openaiAPIKey = os.Getenv("OPENAI_API_KEY")

func queryOpenAI(input string) (string, error) {
	url := "https://api.openai.com/v1/chat/completions"
	requestBody, err := json.Marshal(map[string]interface{}{
		"model":       "gpt-4o-mini",
		"messages":    []map[string]interface{}{{"role": "user", "content": fmt.Sprintf("Provide only the CLI command to solve the following. Dont use code block, just the command: %s", input)}},
		"max_tokens":  100,
		"temperature": 0.7,
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", openaiAPIKey))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if choices, ok := result["choices"].([]interface{}); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]interface{}); ok {
			if message, ok := choice["message"].(map[string]interface{}); ok {
				if content, ok := message["content"].(string); ok {
					return content, nil
				}
			}
		}
	}
	return "", fmt.Errorf("unexpected response format")
}

func main() {
	if openaiAPIKey == "" {
		fmt.Fprintf(os.Stderr, "Error: OPENAI_API_KEY environment variable not set\n")
		os.Exit(1)
	}

	app := tview.NewApplication()

	var inputText string
	inputField := tview.NewInputField()

	inputField.
		SetLabel("Write CLI command for: ").
		SetFieldWidth(30).
		SetDoneFunc(func(key tcell.Key) {
			if key == tcell.KeyEnter {
				inputText = inputField.GetText()
				app.Stop()

				command, err := queryOpenAI(inputText)
				if err != nil {
					fmt.Fprintf(os.Stderr, "failed to query openai: %v\n", err)
					os.Exit(1)
				}

				robotgo.TypeStr(command)
			}
		})

	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(inputField, 3, 1, true).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewBox().SetBackgroundColor(tcell.ColorDefault), 0, 1, false).
		AddItem(nil, 0, 1, false)

	if err := app.SetRoot(flex, true).Run(); err != nil {
		panic(err)
	}
}
