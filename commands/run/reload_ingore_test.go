package run

import "testing"

func Test_isIgnoreReload(t *testing.T) {
	type assert struct {
		pattern string
		path    string
		ignore  bool
	}

	asserts := []assert{
		{"a.txt", "a.txt", true},
		{"*.txt", "a.txt", true},
		{"dir/a.txt", "dir/a.txt", true},
		{"dir/*.txt", "dir/a.txt", true},
		{"dir2/a.txt", "dir1/dir2/a.txt", true},
		{"dir3/a.txt", "dir1/dir2/dir3/a.txt", true},
		{"a.txt", "dir/a.txt", true},
		{"*.txt", "dir/a.txt", true},
		{"a.txt", "dir1/dir2/a.txt", true},
		{"dir2/a.txt", "dir1/dir2/a.txt", true},
		{"dir", "dir", true},
		{"dir/", "dir", false},
		{"dir1/dir2/", "dir1/dir2", false},
		{"/a.txt", "a.txt", true},
		{"/dir/a.txt", "dir/a.txt", true},
		{"/dir1/a.txt", "dir/dir1/a.txt", false},
		{"/a.txt", "dir/a.txt", false},
		{"a.txt", "dir/b.txt", false},
		{"*.txt", "dir/b.txt", true},
		{"dir/*.txt", "dir/b.txt", true},
		{"dir/*.txt", "dir/dir2/b.txt", false},
		{"!b.txt", "b.txt", false},
		{"!b.txt", "dir/b.txt", false},
		{"!/b.txt", "dir/b.txt", false},
		{"/*.txt", "dir/b.txt", false},
	}

	for _, item := range asserts {
		ignore, err := isIgnoreReload(item.pattern, item.path)
		if err != nil {
			t.Error(err)
		}
		if ignore != item.ignore {
			t.Errorf("Match fail: pattern=%s, path=%s", item.pattern, item.path)
		}
	}
}
