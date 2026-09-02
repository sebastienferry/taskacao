package db_test

import (
	"testing"

	"tasks/internal/db"
	"tasks/internal/models"
)

func TestRewriteStorySkillTemplate(t *testing.T) {
	// 1. Verify StageSkillByID lookup for rewrite_story and its aliases
	aliases := []string{"rewrite_story", "rewrite-story", "rewrite"}
	for _, alias := range aliases {
		skill, ok := db.StageSkillByID(alias)
		if !ok {
			t.Errorf("Expected StageSkillByID(%q) to be found", alias)
			continue
		}
		if skill.ID != "rewrite_story" {
			t.Errorf("Expected skill ID 'rewrite_story' for alias %q, got %q", alias, skill.ID)
		}
		if skill.Command != "/rewrite-story" {
			t.Errorf("Expected command '/rewrite-story', got %q", skill.Command)
		}
	}

	// 2. Verify SkillDirNames mapping
	dirName, ok := models.SkillDirNames["rewrite_story"]
	if !ok || dirName != "rewrite-story" {
		t.Errorf("Expected SkillDirNames['rewrite_story'] to be 'rewrite-story', got %q (ok=%v)", dirName, ok)
	}

	// 3. Verify ProjectSkillTemplates contains rewrite_story
	templates := db.ProjectSkillTemplates("speckit")
	found := false
	for _, tmpl := range templates {
		if tmpl.ID == "rewrite_story" {
			found = true
			if tmpl.DirName != "rewrite-story" {
				t.Errorf("Expected template DirName 'rewrite-story', got %q", tmpl.DirName)
			}
			if tmpl.Content == "" {
				t.Errorf("Expected non-empty template content for rewrite_story")
			}
			break
		}
	}
	if !found {
		t.Errorf("Expected ProjectSkillTemplates to include 'rewrite_story'")
	}
}

func TestRefineMacroSkillTemplate(t *testing.T) {
	// 1. Verify StageSkillByID lookup for refine_macro and its aliases
	aliases := []string{"refine_macro", "refine-macro", "refine"}
	for _, alias := range aliases {
		skill, ok := db.StageSkillByID(alias)
		if !ok {
			t.Errorf("Expected StageSkillByID(%q) to be found", alias)
			continue
		}
		if skill.ID != "refine_macro" {
			t.Errorf("Expected skill ID 'refine_macro' for alias %q, got %q", alias, skill.ID)
		}
		if skill.Command != "/refine-macro" {
			t.Errorf("Expected command '/refine-macro', got %q", skill.Command)
		}
		if !skill.Interactive {
			t.Errorf("Expected skill.Interactive to be true for alias %q", alias)
		}
	}

	// 2. Verify SkillDirNames mapping
	dirName, ok := models.SkillDirNames["refine_macro"]
	if !ok || dirName != "refine-macro" {
		t.Errorf("Expected SkillDirNames['refine_macro'] to be 'refine-macro', got %q (ok=%v)", dirName, ok)
	}

	// 3. Verify ProjectSkillTemplates contains refine_macro and interactive steps
	for _, fw := range []string{"speckit", "openspec"} {
		templates := db.ProjectSkillTemplates(fw)
		found := false
		for _, tmpl := range templates {
			if tmpl.ID == "refine_macro" {
				found = true
				if tmpl.DirName != "refine-macro" {
					t.Errorf("Expected template DirName 'refine-macro', got %q", tmpl.DirName)
				}
				if tmpl.Content == "" {
					t.Errorf("Expected non-empty template content for refine_macro (%s)", fw)
				}
				break
			}
		}
		if !found {
			t.Errorf("Expected ProjectSkillTemplates(%s) to include 'refine_macro'", fw)
		}
	}
}
