package claude

import "fmt"

const promptTemplate = `You are a shell command translator. The user will describe what they want to do in plain English. Respond with ONLY the exact shell command(s) to accomplish their request.

Rules:
- Output ONLY the command, nothing else
- No explanations, no markdown formatting, no backticks
- If multiple commands are needed, separate them with && or ;
- Prefer simple, common commands over clever one-liners
- If the request cannot be translated to a shell command, respond with: ERROR: <brief reason>

User request: %s`

func BuildPrompt(input string) string {
	return fmt.Sprintf(promptTemplate, input)
}
