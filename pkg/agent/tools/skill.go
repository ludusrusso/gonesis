package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/ludusrusso/wildgecu/pkg/provider/tool"
	"github.com/ludusrusso/wildgecu/pkg/skill"
)

// SkillTools returns the list_skills and read_skill tools bound to skillsDir.
// Returns an empty slice if skillsDir is empty.
func SkillTools(skillsDir string) []tool.Tool {
	if skillsDir == "" {
		return nil
	}
	return []tool.Tool{
		newListSkillsTool(skillsDir),
		newReadSkillTool(skillsDir),
	}
}

// --- list_skills ---

type listSkillsInput struct{}

type listSkillsOutput struct {
	Skills []skillSummary `json:"skills"`
}

type skillSummary struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Tags        []string `json:"tags,omitempty"`
}

func newListSkillsTool(skillsDir string) tool.Tool {
	return tool.NewTool("list_skills", "List available domain-specific skills with their names, descriptions, and tags.",
		func(ctx context.Context, in listSkillsInput) (listSkillsOutput, error) {
			skills, _ := skill.LoadAll(skillsDir)
			summaries := make([]skillSummary, 0, len(skills))
			for _, s := range skills {
				summaries = append(summaries, skillSummary{
					Name:        s.Name,
					Description: s.Description,
					Tags:        s.Tags,
				})
			}
			return listSkillsOutput{Skills: summaries}, nil
		},
	)
}

// --- read_skill ---

type readSkillInput struct {
	Name string `json:"name" description:"Name of the skill to load"`
}

type readSkillOutput struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

func newReadSkillTool(skillsDir string) tool.Tool {
	return tool.NewTool("read_skill", "Load a specific skill's full content by name.",
		func(ctx context.Context, in readSkillInput) (readSkillOutput, error) {
			if strings.TrimSpace(in.Name) == "" {
				return readSkillOutput{}, fmt.Errorf("name is required")
			}
			s, err := skill.Load(skillsDir, in.Name)
			if err != nil {
				return readSkillOutput{}, fmt.Errorf("loading skill %q: %w", in.Name, err)
			}
			return readSkillOutput{
				Name:    s.Name,
				Content: s.Content,
			}, nil
		},
	)
}
