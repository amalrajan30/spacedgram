package llm

import (
	"context"
	"encoding/json"
	"log"

	"github.com/invopop/jsonschema"
	"github.com/openai/openai-go"
)

type ClazeQuestion struct {
	Question string `json:"question" jsonschema_description:"Fill in the blank type question from the notes"`
	Answer   string `json:"answer" jsonschema_description:"Answer of the fill in the blank type question"`
}

func GenerateSchema[T any]() interface{} {
	// Structured Outputs uses a subset of JSON schema
	// These flags are necessary to comply with the subset
	reflector := jsonschema.Reflector{
		AllowAdditionalProperties: false,
		DoNotReference:            true,
	}
	var v T
	schema := reflector.Reflect(v)
	return schema
}

// Generate the JSON schema at initialization time
var ClazeQuestionResponseSchema = GenerateSchema[ClazeQuestion]()

func GenerateClaseQuestionAnswer(question string) (*ClazeQuestion, error) {
	log.Println("Generating claze question using gpt")
	client := openai.NewClient()
	ctx := context.Background()

	schemaParam := openai.ResponseFormatJSONSchemaJSONSchemaParam{
		Name:        openai.F("create_question"),
		Description: openai.F("Generate claze type question and answer from a piece of information"),
		Schema:      openai.F(ClazeQuestionResponseSchema),
		Strict:      openai.Bool(true),
	}

	// Query the Chat Completions API
	chat, err := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage("You are a specialized educational assistant designed to create fill-in-the-blank questions from provided text content."),
			openai.UserMessage(question),
		}),
		ResponseFormat: openai.F[openai.ChatCompletionNewParamsResponseFormatUnion](
			openai.ResponseFormatJSONSchemaParam{
				Type:       openai.F(openai.ResponseFormatJSONSchemaTypeJSONSchema),
				JSONSchema: openai.F(schemaParam),
			},
		),
		// Only certain models can perform structured outputs
		Model: openai.F(openai.ChatModelGPT4oMini2024_07_18),
	})

	if err != nil {
		log.Printf("failed to generate claze question: %v", err)
		return nil, err
	}

	clazeQuestionAnswer := ClazeQuestion{}
	err = json.Unmarshal([]byte(chat.Choices[0].Message.Content), &clazeQuestionAnswer)
	if err != nil {
		log.Printf("failed to parse llm response %v", err)
		return nil, err
	}

	return &clazeQuestionAnswer, nil
}
