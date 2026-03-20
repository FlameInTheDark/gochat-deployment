package deployer

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"

	cli "github.com/urfave/cli/v3"
)

const sfuServiceType = "sfu"

type jwtHeader struct {
	Algorithm string `json:"alg"`
	Type      string `json:"typ"`
}

type serviceTokenClaims struct {
	Type string `json:"typ"`
	ID   string `json:"id"`
}

func tokensCommand() *cli.Command {
	return &cli.Command{
		Name:  "tokens",
		Usage: "Generate helper tokens for manually managed services",
		Commands: []*cli.Command{
			sfuTokenCommand(),
		},
	}
}

func sfuTokenCommand() *cli.Command {
	return &cli.Command{
		Name:  "sfu",
		Usage: "Generate the webhook token used by an external SFU node",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "secret", Aliases: []string{"s"}, Usage: "Webhook JWT secret used by the webhook service", Required: true},
			&cli.StringFlag{Name: "id", Aliases: []string{"i"}, Usage: "SFU service id. If empty, a UUIDv4 is generated."},
			&cli.StringFlag{Name: "format", Aliases: []string{"f"}, Value: "text", Usage: "Output format: text or json"},
			&cli.BoolFlag{Name: "header", Usage: "Include the X-Webhook-Token header form in the output"},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			serviceID := cmd.String("id")
			if serviceID == "" {
				generatedID, err := newUUIDv4()
				if err != nil {
					return err
				}
				serviceID = generatedID
			}

			token, err := generateServiceToken(cmd.String("secret"), sfuServiceType, serviceID)
			if err != nil {
				return err
			}

			return writeSFUTokenOutput(cmd.Writer, cmd.String("format"), serviceID, token, cmd.Bool("header"))
		},
	}
}

func generateServiceToken(secret, serviceType, serviceID string) (string, error) {
	if secret == "" {
		return "", fmt.Errorf("secret is required")
	}
	if serviceType == "" {
		return "", fmt.Errorf("service type is required")
	}
	if serviceID == "" {
		return "", fmt.Errorf("service id is required")
	}

	headerPart, err := encodeJWTPart(jwtHeader{
		Algorithm: "HS256",
		Type:      "JWT",
	})
	if err != nil {
		return "", err
	}
	claimsPart, err := encodeJWTPart(serviceTokenClaims{
		Type: serviceType,
		ID:   serviceID,
	})
	if err != nil {
		return "", err
	}

	signingInput := headerPart + "." + claimsPart
	mac := hmac.New(sha256.New, []byte(secret))
	if _, err := mac.Write([]byte(signingInput)); err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}

	return signingInput + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}

func encodeJWTPart(value any) (string, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("marshal jwt part: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(payload), nil
}

func newUUIDv4() (string, error) {
	data := make([]byte, 16)
	if _, err := rand.Read(data); err != nil {
		return "", fmt.Errorf("generate uuid: %w", err)
	}

	data[6] = (data[6] & 0x0f) | 0x40
	data[8] = (data[8] & 0x3f) | 0x80

	return fmt.Sprintf(
		"%02x%02x%02x%02x-%02x%02x-%02x%02x-%02x%02x-%02x%02x%02x%02x%02x%02x",
		data[0], data[1], data[2], data[3],
		data[4], data[5],
		data[6], data[7],
		data[8], data[9],
		data[10], data[11], data[12], data[13], data[14], data[15],
	), nil
}

func writeSFUTokenOutput(output io.Writer, format, serviceID, token string, includeHeader bool) error {
	if output == nil {
		output = io.Discard
	}

	headerValue := ""
	if includeHeader {
		headerValue = "X-Webhook-Token: " + token
	}

	switch format {
	case "", "text":
		fmt.Fprintf(output, "service_id=%s\n", serviceID)
		fmt.Fprintf(output, "token=%s\n", token)
		if headerValue != "" {
			fmt.Fprintln(output, headerValue)
		}
		return nil
	case "json":
		payload := struct {
			ServiceID string `json:"service_id"`
			Token     string `json:"token"`
			Header    string `json:"header,omitempty"`
		}{
			ServiceID: serviceID,
			Token:     token,
			Header:    headerValue,
		}
		encoded, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal output: %w", err)
		}
		fmt.Fprintln(output, string(encoded))
		return nil
	default:
		return fmt.Errorf("unsupported output format %q", format)
	}
}
