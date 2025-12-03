package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/mail"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
)



type FeedbackForm struct {
	Metadata struct {
		Arch          string    `json:"arch"`
		Os            string    `json:"os"`
		TimeCreated   time.Time `json:"timeCreated"`
		TimeSubmitted time.Time `json:"timeSubmitted"`
	}
	FeedbackType string `json:"feedbackType"`
	Name         string `json:"name"`
	Email        string `json:"email"`
	Message      string `json:"message"`
}

func AskForFeedback(accessible bool) error {

	formData := &FeedbackForm{
		Metadata: struct {
			Arch          string    `json:"arch"`
			Os            string    `json:"os"`
			TimeCreated   time.Time `json:"timeCreated"`
			TimeSubmitted time.Time `json:"timeSubmitted"`
		}{
			Arch:        runtime.GOARCH,
			Os:          runtime.GOOS,
			TimeCreated: time.Now(),
		},
	}

	submitForm := false

	form := huh.NewForm(
		huh.NewGroup(
			// Ask the user to categorize their feedback
			huh.NewSelect[string]().
				Title("What is the nature of your feedback?").
				Options(
					huh.NewOption("Bug Report", "bug"),
					huh.NewOption("Feature Request", "enhancement"),
					huh.NewOption("Comment", "comment"),
				).
				Value(&formData.FeedbackType),
		),
		huh.NewGroup(
			huh.NewInput().
				Title("What’s your name?").
				Value(&formData.Name).
				// Validating fields is easy. The form will mark erroneous fields
				// and display error messages accordingly.
				Validate(func(str string) error {
					if len(str) == 0 {
						return errors.New("Sorry, `name` is a required field.")
					}
					return nil
				}),
			huh.NewInput().
				Title("What’s your email?").
				Value(&formData.Email).
				// Validating fields is easy. The form will mark erroneous fields
				// and display error messages accordingly.
				Validate(func(str string) error {
					if len(str) == 0 {
						return errors.New("Sorry, `email` is a required field.")
					}

					if _, err := mail.ParseAddress(strings.ToLower(str)); err != nil {
						return errors.New("You must supply a valid email.")
					}

					return nil
				}),
			huh.NewText().
				Title("What do you want to say?").
				CharLimit(1000).
				Value(&formData.Message),

			huh.NewConfirm().
				Title("Are you ready to submit your feedback?").
				Value(&submitForm),
		),
	)

	// To allow users with screen readers to use feedback
	form.WithAccessible(accessible)

	if err := form.Run(); err != nil {
		return err
	}

	if submitForm {
		formData.Metadata.TimeSubmitted = time.Now()
		formBody, err := json.Marshal(formData)
		if err != nil {
			return fmt.Errorf("unable to marshal form data. %w", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(time.Second*10))
		defer cancel()

		// This is my personal website running Laravel. I already have it up and configured for other projects.
		feedbackUrl := "https://artsie.red/api/zvm-feedback"
		if fUrl := os.Getenv("ZVM_FEEDBACK_URL"); fUrl != "" {
			feedbackUrl = fUrl
		}

		req, err := http.NewRequest("POST", feedbackUrl, bytes.NewBuffer(formBody))
		if err != nil {
			return err
		}

		req.WithContext(ctx)
		req.Header.Set("Content-Type", "application/json")

		c := http.DefaultClient
		resp, err := c.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
			return fmt.Errorf("error submitting your feedback: (%d) %s", resp.StatusCode, string(body))
		}

		fmt.Println("✅ Feedback submitted successfully. Thank you for taking the time to write it! :)")
	}

	return nil
}
