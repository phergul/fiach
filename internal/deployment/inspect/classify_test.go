package inspect

import "testing"

func TestClassifyRelativePath(t *testing.T) {
	t.Parallel()

	cases := []struct {
		path string
		want fileClass
	}{
		{path: "Data/plugin.dll", want: fileClassPE},
		{path: "bin/Game.exe", want: fileClassPE},
		{path: "textures/icon.png", want: fileClassImage},
		{path: "config/settings.ini", want: fileClassText},
		{path: "mods/archive.zip", want: fileClassArchive},
		{path: "unknown/file.bin", want: fileClassBinary},
	}

	for _, testCase := range cases {
		if got := classifyRelativePath(testCase.path); got != testCase.want {
			t.Fatalf("classifyRelativePath(%q) = %q, want %q", testCase.path, got, testCase.want)
		}
	}
}

func TestCleanArchiveEntryNameRejectsParentTraversal(t *testing.T) {
	t.Parallel()

	if _, err := cleanArchiveEntryName("../outside.txt"); err == nil {
		t.Fatal("cleanArchiveEntryName() error = nil, want traversal rejection")
	}
}
