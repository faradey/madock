package paths

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsFileExist(t *testing.T) {
	// Create a temp file for testing
	tmpFile, err := os.CreateTemp("", "testfile")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "existing file",
			path:     tmpPath,
			expected: true,
		},
		{
			name:     "non-existing file",
			path:     "/this/path/does/not/exist/12345.txt",
			expected: false,
		},
		{
			name:     "empty path",
			path:     "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsFileExist(tt.path)
			if got != tt.expected {
				t.Errorf("IsFileExist(%q) = %v, want %v", tt.path, got, tt.expected)
			}
		})
	}
}

func TestIsFileExistDirectory(t *testing.T) {
	// Test with existing directory
	tmpDir, err := os.MkdirTemp("", "testdir")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// IsFileExist should return true for directories too
	if !IsFileExist(tmpDir) {
		t.Errorf("IsFileExist(%q) = false for existing directory, want true", tmpDir)
	}
}

func TestGetExecDirNameByPath(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"/home/user/projects/madock", "madock"},
		{"/var/www/html", "html"},
		{"/single", "single"},
		{"relative/path", "path"},
		{"", "."},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := GetExecDirNameByPath(tt.path)
			if got != tt.expected {
				t.Errorf("GetExecDirNameByPath(%q) = %q, want %q", tt.path, got, tt.expected)
			}
		})
	}
}

func TestGetRunDirPath(t *testing.T) {
	// Get the current working directory
	expected, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	got := GetRunDirPath()
	if got != expected {
		t.Errorf("GetRunDirPath() = %q, want %q", got, expected)
	}
}

func TestGetRunDirName(t *testing.T) {
	expected := filepath.Base(GetRunDirPath())
	got := GetRunDirName()
	if got != expected {
		t.Errorf("GetRunDirName() = %q, want %q", got, expected)
	}
}

func TestSameFile(t *testing.T) {
	// Create a temp file
	tmpFile, err := os.CreateTemp("", "samefile")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	// Create another temp file
	tmpFile2, err := os.CreateTemp("", "samefile2")
	if err != nil {
		t.Fatalf("Failed to create temp file 2: %v", err)
	}
	tmpPath2 := tmpFile2.Name()
	tmpFile2.Close()
	defer os.Remove(tmpPath2)

	tests := []struct {
		name     string
		a        string
		b        string
		expected bool
		hasError bool
	}{
		{
			name:     "same path",
			a:        tmpPath,
			b:        tmpPath,
			expected: true,
			hasError: false,
		},
		{
			name:     "different files",
			a:        tmpPath,
			b:        tmpPath2,
			expected: false,
			hasError: false,
		},
		{
			name:     "first file doesn't exist",
			a:        "/nonexistent/file.txt",
			b:        tmpPath,
			expected: false,
			hasError: false,
		},
		{
			name:     "second file doesn't exist",
			a:        tmpPath,
			b:        "/nonexistent/file.txt",
			expected: false,
			hasError: false,
		},
		{
			name:     "both files don't exist",
			a:        "/nonexistent/file1.txt",
			b:        "/nonexistent/file2.txt",
			expected: false,
			hasError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SameFile(tt.a, tt.b)
			if tt.hasError && err == nil {
				t.Errorf("SameFile(%q, %q) expected error, got nil", tt.a, tt.b)
			}
			if !tt.hasError && err != nil {
				t.Errorf("SameFile(%q, %q) unexpected error: %v", tt.a, tt.b, err)
			}
			if got != tt.expected {
				t.Errorf("SameFile(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.expected)
			}
		})
	}
}

func TestCopy(t *testing.T) {
	// Create source file with content
	srcFile, err := os.CreateTemp("", "copytest_src")
	if err != nil {
		t.Fatalf("Failed to create source temp file: %v", err)
	}
	srcPath := srcFile.Name()
	testContent := "Hello, World! Test content for copy."
	srcFile.WriteString(testContent)
	srcFile.Close()
	defer os.Remove(srcPath)

	// Create destination path
	dstPath := srcPath + "_copy"
	defer os.Remove(dstPath)

	// Perform copy
	err = Copy(srcPath, dstPath)
	if err != nil {
		t.Fatalf("Copy(%q, %q) failed: %v", srcPath, dstPath, err)
	}

	// Verify destination exists
	if !IsFileExist(dstPath) {
		t.Errorf("Copy did not create destination file")
	}

	// Verify content
	content, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("Failed to read destination file: %v", err)
	}
	if string(content) != testContent {
		t.Errorf("Copy content mismatch: got %q, want %q", string(content), testContent)
	}
}

func TestCopyNonExistentSource(t *testing.T) {
	err := Copy("/nonexistent/source.txt", "/tmp/dest.txt")
	if err == nil {
		t.Error("Copy with non-existent source should return error")
	}
}

func TestGetDirs(t *testing.T) {
	// Create a temp directory structure
	tmpDir, err := os.MkdirTemp("", "getdirs")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create subdirectories
	os.Mkdir(filepath.Join(tmpDir, "dir1"), 0755)
	os.Mkdir(filepath.Join(tmpDir, "dir2"), 0755)
	os.Mkdir(filepath.Join(tmpDir, "dir3"), 0755)

	// Create a file (should not be included)
	os.WriteFile(filepath.Join(tmpDir, "file.txt"), []byte("test"), 0644)

	dirs := GetDirs(tmpDir)

	if len(dirs) != 3 {
		t.Errorf("GetDirs returned %d dirs, want 3", len(dirs))
	}

	// Check that all directories are present
	expected := map[string]bool{"dir1": true, "dir2": true, "dir3": true}
	for _, dir := range dirs {
		if !expected[dir] {
			t.Errorf("Unexpected directory: %s", dir)
		}
		delete(expected, dir)
	}
	if len(expected) > 0 {
		t.Errorf("Missing directories: %v", expected)
	}
}

func TestGetFiles(t *testing.T) {
	// Create a temp directory structure
	tmpDir, err := os.MkdirTemp("", "getfiles")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create files
	os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "file2.txt"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "file3.txt"), []byte("test"), 0644)

	// Create a directory (should not be included)
	os.Mkdir(filepath.Join(tmpDir, "subdir"), 0755)

	files := GetFiles(tmpDir)

	if len(files) != 3 {
		t.Errorf("GetFiles returned %d files, want 3", len(files))
	}

	// Check that all files are present
	expected := map[string]bool{"file1.txt": true, "file2.txt": true, "file3.txt": true}
	for _, file := range files {
		if !expected[file] {
			t.Errorf("Unexpected file: %s", file)
		}
		delete(expected, file)
	}
	if len(expected) > 0 {
		t.Errorf("Missing files: %v", expected)
	}
}

func TestGetFilesRecursively(t *testing.T) {
	// Create a temp directory structure
	tmpDir, err := os.MkdirTemp("", "getfilesrecur")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create nested structure
	os.Mkdir(filepath.Join(tmpDir, "level1"), 0755)
	os.Mkdir(filepath.Join(tmpDir, "level1", "level2"), 0755)

	os.WriteFile(filepath.Join(tmpDir, "root.txt"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "level1", "l1.txt"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "level1", "level2", "l2.txt"), []byte("test"), 0644)

	files := GetFilesRecursively(tmpDir)

	if len(files) != 3 {
		t.Errorf("GetFilesRecursively returned %d files, want 3", len(files))
	}
}

func TestMakeDirsByPath(t *testing.T) {
	// Create a temp directory
	tmpDir, err := os.MkdirTemp("", "makedirs")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Test creating nested directories
	newPath := tmpDir + "/new/nested/path"
	result := MakeDirsByPath(newPath)

	if result != newPath {
		t.Errorf("MakeDirsByPath returned %q, want %q", result, newPath)
	}

	if !IsFileExist(newPath) {
		t.Errorf("MakeDirsByPath did not create the directory")
	}
}

func TestMakeDirsByPathEmpty(t *testing.T) {
	// Test with empty path
	result := MakeDirsByPath("")
	if result != "" {
		t.Errorf("MakeDirsByPath(\"\") = %q, want empty string", result)
	}
}
