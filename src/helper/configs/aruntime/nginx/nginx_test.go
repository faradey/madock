package nginx

import "testing"

func TestGetMaxPort(t *testing.T) {
	tests := []struct {
		name     string
		conf     map[string]string
		expected int
	}{
		{
			name:     "empty config",
			conf:     map[string]string{},
			expected: 0,
		},
		{
			name: "single project",
			conf: map[string]string{
				"project1": "1",
			},
			expected: 1,
		},
		{
			name: "multiple sequential projects",
			conf: map[string]string{
				"project1": "1",
				"project2": "2",
				"project3": "3",
			},
			expected: 3,
		},
		{
			name: "gap in ports",
			conf: map[string]string{
				"project1": "1",
				"project3": "3",
			},
			// Function finds first available slot (2-1=1), so returns 1
			expected: 1,
		},
		{
			name: "non-sequential ports",
			conf: map[string]string{
				"project1": "1",
				"project2": "2",
				"project5": "5",
			},
			// Gap at 3, so returns 2
			expected: 2,
		},
		{
			name: "starting from higher port",
			conf: map[string]string{
				"project5": "5",
			},
			// First available is 1-1=0
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getMaxPort(tt.conf)
			if got != tt.expected {
				t.Errorf("getMaxPort() = %d, want %d", got, tt.expected)
			}
		})
	}
}
