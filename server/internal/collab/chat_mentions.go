package collab

import "regexp"

var chatMentionToken = regexp.MustCompile(`@([a-zA-Z0-9_-]+)`)

// ExtractChatMentions returns unique @handles from text (without the @ prefix), order preserved, capped at max.
func ExtractChatMentions(text string, max int) []string {
	if max <= 0 || text == "" {
		return nil
	}
	seen := make(map[string]struct{})
	var out []string
	for _, m := range chatMentionToken.FindAllStringSubmatch(text, -1) {
		if len(m) < 2 {
			continue
		}
		name := m[1]
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
		if len(out) >= max {
			break
		}
	}
	return out
}
