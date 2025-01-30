package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/go-vgo/robotgo"
	"github.com/rivo/tview"
)

var openaiAPIKey = os.Getenv("OPENAI_API_KEY")
var historyFilePath = "/tmp/kk-history.jsonl" // Global variable for history file path

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

func savePromptToHistory(prompt string) error {
	if prompt == "" {
		return nil // Do not save empty prompts
	}

	data, err := os.ReadFile(historyFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			// If the file does not exist, create it
			file, err := os.Create(historyFilePath)
			if err != nil {
				return err
			}
			file.Close()
			data = []byte{}
		} else {
			return err
		}
	}

	lines := strings.Split(string(data), "\n")
	var prompts []string
	for _, line := range lines {
		if line == "" {
			continue
		}
		var promptData map[string]string
		if err := json.Unmarshal([]byte(line), &promptData); err != nil {
			return err
		}
		if existingPrompt, ok := promptData["prompt"]; ok && existingPrompt == prompt {
			return nil // Do not save if prompt already exists
		}
		prompts = append(prompts, line)
	}

	// Add the new prompt
	promptData := map[string]string{"prompt": prompt}
	jsonData, err := json.Marshal(promptData)
	if err != nil {
		return err
	}
	prompts = append(prompts, string(jsonData))

	// Keep only the last 10 prompts
	if len(prompts) > 10 {
		prompts = prompts[len(prompts)-10:]
	}

	// Write back to the file
	file, err := os.OpenFile(historyFilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	for _, p := range prompts {
		if _, err := file.WriteString(p + "\n"); err != nil {
			return err
		}
	}

	return nil
}

func getLastNPrompts(n int) ([]string, error) {
	data, err := os.ReadFile(historyFilePath)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(data), "\n")
	var prompts []string
	for _, line := range lines {
		if line == "" {
			continue
		}
		var promptData map[string]string
		if err := json.Unmarshal([]byte(line), &promptData); err != nil {
			return nil, err
		}
		if prompt, ok := promptData["prompt"]; ok {
			prompts = append(prompts, prompt)
		}
	}

	if len(prompts) > n {
		prompts = prompts[len(prompts)-n:] // Get the last n prompts
	}

	return prompts, nil
}

func main() {
	if openaiAPIKey == "" {
		fmt.Fprintf(os.Stderr, "Error: OPENAI_API_KEY environment variable not set\n")
		os.Exit(1)
	}

	// Ensure the prompt history file exists
	if _, err := os.Stat(historyFilePath); os.IsNotExist(err) {
		file, err := os.Create(historyFilePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to create prompt history file: %v\n", err)
			os.Exit(1)
		}
		file.Close()
	}

	app := tview.NewApplication()

	var inputText string
	inputField := tview.NewInputField()

	// Create a TextView for the "Esc to close" message
	escTextView := tview.NewTextView().
		SetText("Esc to close").
		SetTextColor(tcell.ColorGray)

	// Load the last 10 prompts
	lastPrompts, err := getLastNPrompts(10)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load prompt history: %v\n", err)
	}

	currentPromptIndex := len(lastPrompts) // Start at the end of the list

	inputField.
		SetLabel("Write CLI command for: ").
		SetFieldWidth(80).
		SetPlaceholder("Press up/down to navigate history").
		SetFieldBackgroundColor(tcell.ColorLightBlue).
		SetFieldTextColor(tcell.ColorBlack).
		SetPlaceholderTextColor(tcell.ColorBlack).
		SetDoneFunc(func(key tcell.Key) {
			if key == tcell.KeyEnter {
				inputText = inputField.GetText()
				app.Stop()

				if err := savePromptToHistory(inputText); err != nil {
					fmt.Fprintf(os.Stderr, "failed to save prompt to history: %v\n", err)
				}

				command, err := queryOpenAI(inputText)
				if err != nil {
					fmt.Fprintf(os.Stderr, "failed to query openai: %v\n", err)
					os.Exit(1)
				}

				robotgo.TypeStr(command)
			}
		}).
		SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			switch event.Key() {
			case tcell.KeyUp:
				if currentPromptIndex > 0 {
					currentPromptIndex--
					inputField.SetText(lastPrompts[currentPromptIndex])
				}
				return nil
			case tcell.KeyDown:
				if currentPromptIndex < len(lastPrompts)-1 {
					currentPromptIndex++
					inputField.SetText(lastPrompts[currentPromptIndex])
				} else {
					currentPromptIndex = len(lastPrompts) // Reset to allow new input
					inputField.SetText("")
				}
				return nil
			case tcell.KeyEscape:
				app.Stop()
				return nil
			}
			return event
		})

	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(inputField, 1, 1, true).  // Adjusted height for input field
		AddItem(escTextView, 1, 0, false) // Place the TextView directly below

	app.SetFocus(inputField)
	app.SetRoot(flex, true)

	if err := app.Run(); err != nil {
		panic(err)
	}
}
