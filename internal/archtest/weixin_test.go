package archtest

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Suki is the product, so a mini program message with no face on it is a bug, and so is a
// mood with no picture. These read the source, because CI has no WeChat DevTools.

func read(t *testing.T, parts ...string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(parts...))
	if err != nil {
		t.Fatal(err)
	}
	return strings.ReplaceAll(string(raw), "\r\n", "\n")
}

func TestEveryMoodHasAFace(t *testing.T) {
	r := root(t)
	lib := read(t, r, "weixin", "lib", "suki.js")
	list := regexp.MustCompile(`FACES = \[([^\]]*)\]`).FindStringSubmatch(lib)
	if list == nil {
		t.Fatal("weixin/lib/suki.js has no list of faces")
	}
	faces := map[string]bool{}
	for _, m := range regexp.MustCompile(`'(\w+)'`).FindAllStringSubmatch(list[1], -1) {
		faces[m[1]] = true
	}
	// Every face the app has, and every mood the app can be told.
	want := []string{"portrait", "chibi"}
	types := read(t, r, "mobile", "src", "api", "types.ts")
	mood := regexp.MustCompile(`type Mood = ([^;\n]*)`).FindStringSubmatch(types)
	if mood == nil {
		t.Fatal("mobile/src/api/types.ts has no Mood type")
	}
	for _, m := range regexp.MustCompile(`'(\w+)'`).FindAllStringSubmatch(mood[1], -1) {
		want = append(want, m[1])
	}
	for _, name := range want {
		if !faces[name] {
			t.Errorf("mood %q has no face in weixin/lib/suki.js", name)
		}
	}
	for name := range faces {
		if _, err := os.Stat(filepath.Join(r, "weixin", "assets", "suki", name+".jpg")); err != nil {
			t.Errorf("face %q has no file: %v", name, err)
		}
	}
}

func TestEverySukiMessageHasAnAvatar(t *testing.T) {
	r := root(t)
	wxml := read(t, r, "weixin", "pages", "suki", "suki.wxml")
	row := regexp.MustCompile(`(?s)<view wx:else class="suki-row">(.*?)<view class="suki-col">`).FindStringSubmatch(wxml)
	if row == nil {
		t.Fatal("pages/suki/suki.wxml: Suki's messages no longer use the suki-row with an avatar beside the bubble")
	}
	if !strings.Contains(row[1], `<suki-avatar`) || !strings.Contains(row[1], `mood="{{item.mood}}"`) || !strings.Contains(row[1], `mood-avatar="{{moodAvatar}}"`) {
		t.Errorf("a Suki message must show her avatar with its mood and the Avatar moods setting: %s", row[1])
	}
	// Nothing but the person's own messages and event notes may skip the row.
	if n := len(regexp.MustCompile(`class="bubble[ "]`).FindAllString(wxml, -1)); n != 3 {
		t.Errorf("pages/suki/suki.wxml has %d bubbles, expected the person's, Suki's and the typing one", n)
	}
	if !strings.Contains(wxml, `<suki-avatar mood="{{headerMood}}"`) {
		t.Error("the Suki page's header has lost her avatar")
	}
	if js := read(t, r, "weixin", "pages", "suki", "suki.js"); !strings.Contains(js, "meta.mood") {
		t.Error("pages/suki/suki.js no longer reads the message's mood")
	}
}

func TestTabBarAndPackageFitWeChatsLimits(t *testing.T) {
	r := root(t)
	var app struct {
		TabBar struct {
			List []struct{ IconPath, SelectedIconPath string } `json:"list"`
		} `json:"tabBar"`
	}
	if err := json.Unmarshal([]byte(read(t, r, "weixin", "app.json")), &app); err != nil {
		t.Fatal(err)
	}
	for _, tab := range app.TabBar.List {
		for _, icon := range []string{tab.IconPath, tab.SelectedIconPath} {
			if icon == "" {
				t.Error("a tab has no icon")
				continue
			}
			info, err := os.Stat(filepath.Join(r, "weixin", filepath.FromSlash(icon)))
			if err != nil {
				t.Error(err)
			} else if info.Size() > 40*1024 {
				t.Errorf("%s is %d bytes, over WeChat's 40 KB tab icon limit", icon, info.Size())
			}
		}
	}
	var total int64
	err := filepath.WalkDir(filepath.Join(r, "weixin"), func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && d.Name() == "e2e" {
			return filepath.SkipDir
		}
		if !d.IsDir() {
			info, err := d.Info()
			if err != nil {
				return err
			}
			total += info.Size()
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if total > 1500*1024 {
		t.Errorf("the mini program is %d bytes, too close to WeChat's 2 MB package limit", total)
	}
	if cfg := read(t, r, "weixin", "project.config.json"); !strings.Contains(cfg, `"value": "e2e"`) {
		t.Error("project.config.json must keep e2e out of the uploaded package")
	}
}
