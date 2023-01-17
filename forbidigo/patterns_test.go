package forbidigo

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestParseValidPatterns(t *testing.T) {
	for _, tc := range []struct {
		name            string
		ptrn            string
		expectedComment string
		expectedPattern string
		expectedPackage string
		expectedFile    []string
	}{
		{
			name: "simple expression, no comment",
			ptrn: `fmt\.Errorf`,
		},
		{
			name: "anchored expression, no comment",
			ptrn: `^fmt\.Errorf$`,
		},
		{
			name:            "contains multiple subexpression, with comment",
			ptrn:            `(f)mt\.Errorf(# a comment)?`,
			expectedComment: "a comment",
		},
		{
			name:            "simple expression with comment",
			ptrn:            `fmt\.Println(# Please don't use this!)?`,
			expectedComment: "Please don't use this!",
		},
		{
			name:            "deeply nested expression with comment",
			ptrn:            `fmt\.Println((((# Please don't use this!))))?`,
			expectedComment: "Please don't use this!",
		},
		{
			name:            "anchored expression with comment",
			ptrn:            `^fmt\.Println(# Please don't use this!)?$`,
			expectedComment: "Please don't use this!",
		},
		{
			name:            "match import",
			ptrn:            `{p: "^fmt\\.Println$"}`,
			expectedPattern: `^fmt\.Println$`,
		},
		{
			name: "match import with YAML",
			ptrn: `{msg: hello world,
p: ^fmt\.Println$
}`,
			expectedComment: "hello world",
			expectedPattern: `^fmt\.Println$`,
		},
		{
			name:            "match import with YAML, no line breaks",
			ptrn:            `{p: ^fmt\.Println$}`,
			expectedPattern: `^fmt\.Println$`,
		},
		{
			name: "simple YAML",
			ptrn: `p: ^fmt\.Println$
`,
			expectedPattern: `^fmt\.Println$`,
		},
		{
			name:            "match import with YAML and file filter",
			ptrn:            `{p: ^fmt\.Println$, ignore: ["**", "!**/main.go"]}`,
			expectedPattern: `^fmt\.Println$`,
			expectedFile:    []string{`**`, `!**/main.go`},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ptrn, err := parse(tc.ptrn)
			require.Nil(t, err)
			expectedPattern := tc.expectedPattern
			if expectedPattern == "" {
				expectedPattern = tc.ptrn
			}
			assert.Equal(t, expectedPattern, ptrn.re.String(), "pattern")
			if assert.Equal(t, tc.expectedPackage, ptrn.Package, "package") && tc.expectedPackage != "" {
				assert.Equal(t, tc.expectedPackage, ptrn.pkgRe.String(), "package RE")
			}
			assert.Equal(t, tc.expectedComment, ptrn.Msg, "comment")
		})
	}
}

func TestParseInvalidPattern_ReturnsError(t *testing.T) {
	_, err := parse(`fmt\`)
	assert.NotNil(t, err)
}

func TestUnmarshalYAML(t *testing.T) {
	for _, tc := range []struct {
		name            string
		yaml            string
		expectedErr     string
		expectedComment string
		expectedPattern string
	}{
		{
			name: "string: simple expression, no comment",
			yaml: `fmt\.Errorf`,
		},
		{
			name:            "string: contains multiple subexpression, with comment",
			yaml:            `(f)mt\.Errorf(# a comment)?`,
			expectedComment: "a comment",
		},
		{
			name:            "struct: simple expression, no comment",
			yaml:            `p: fmt\.Errorf`,
			expectedPattern: `fmt\.Errorf`,
		},
		{
			name: "match import with YAML",
			yaml: `p: ^fmt\.Println$
`,
			expectedPattern: `^fmt\.Println$`,
		},
		{
			name:        "string: invalid regexp",
			yaml:        `fmt\`,
			expectedErr: "unable to compile source code pattern `fmt\\`: error parsing regexp: trailing backslash at end of expression: ``",
		},
		{
			name: "struct: invalid regexp",
			yaml: `p: fmt\
`,
			expectedErr: "unable to compile source code pattern `fmt\\`: error parsing regexp: trailing backslash at end of expression: ``",
		},
		{
			name: "invalid struct",
			yaml: `Foo: bar`,
			expectedErr: `pattern is neither a regular expression string (yaml: unmarshal errors:
  line 1: cannot unmarshal !!map into string) nor a Pattern struct (yaml: unmarshal errors:
  line 1: field Foo not found in type forbidigo.pattern)`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var p yamlPattern
			err := yaml.UnmarshalStrict([]byte(tc.yaml), &p)
			if tc.expectedErr == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.Equal(t, tc.expectedErr, err.Error())
				return
			}
			expectedPattern := tc.expectedPattern
			if expectedPattern == "" {
				expectedPattern = tc.yaml
			}
			assert.Equal(t, expectedPattern, p.re.String(), "pattern")
			assert.Equal(t, tc.expectedComment, p.Msg, "comment")
		})
	}
}

func TestGlob(t *testing.T) {
	testcases := map[string]struct {
		ignore           []string
		expectError      bool
		expectIgnoreFile map[string]bool
	}{
		"disable": {
			ignore: []string{"main.go"},
			expectIgnoreFile: map[string]bool{
				"main.go":  true,
				"hello.go": false,
			},
		},
		"enable": {
			ignore: []string{"*", "!main.go"},
			expectIgnoreFile: map[string]bool{
				"main.go":  false,
				"hello.go": true,
			},
		},
		"all": {
			ignore: []string{},
			expectIgnoreFile: map[string]bool{
				"main.go":  false,
				"hello.go": false,
			},
		},
		"double": {
			ignore: []string{"**/main.go"},
			expectIgnoreFile: map[string]bool{
				"example.com/hello/main.go": true,
			},
		},
	}

	for name, tc := range testcases {
		t.Run(name, func(t *testing.T) {
			p := pattern{
				Pattern: "unused",
				Ignore:  tc.ignore,
			}
			if err := p.validate(); err != nil {
				if !tc.expectError {
					t.Fatalf("unexpected validation error: %v", err)
				}
			} else {
				if tc.expectError {
					t.Fatal("validation unexpectedly succeeded")
				}
			}

			for file, expectIgnore := range tc.expectIgnoreFile {
				t.Run(strings.ReplaceAll(file, "/", "-"), func(t *testing.T) {
					ignore := p.ignoreFile(file)
					if expectIgnore && !ignore {
						t.Fatal("expected file to be ignored")
					}
					if !expectIgnore && ignore {
						t.Fatal("expected file not to be ignored")
					}
				})
			}
		})
	}
}
