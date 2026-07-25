package ledger

import "testing"

func TestParseMessage(t *testing.T) {
	cases := []struct {
		name    string
		text    string
		want    Entry
		wantErr error
	}{
		{
			name: "unsigned amount defaults to expense",
			text: "-100 午餐",
			want: Entry{Amount: -100, Category: "午餐"},
		},
		{
			name: "plain number defaults to expense",
			text: "100 午餐",
			want: Entry{Amount: -100, Category: "午餐"},
		},
		{
			name: "explicit plus is income",
			text: "+5000 薪水 七月獎金",
			want: Entry{Amount: 5000, Category: "薪水", Note: "七月獎金"},
		},
		{
			name: "note joins remaining fields",
			text: "-350 交通 捷運 來回",
			want: Entry{Amount: -350, Category: "交通", Note: "捷運 來回"},
		},
		{
			name:    "missing category is not an entry",
			text:    "-100",
			wantErr: ErrNotAnEntry,
		},
		{
			name:    "non-numeric first field is not an entry",
			text:    "help me",
			wantErr: ErrNotAnEntry,
		},
		{
			name:    "empty message is not an entry",
			text:    "",
			wantErr: ErrNotAnEntry,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseMessage(tc.text)
			if tc.wantErr != nil {
				if err != tc.wantErr {
					t.Fatalf("ParseMessage(%q) error = %v, want %v", tc.text, err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseMessage(%q) unexpected error: %v", tc.text, err)
			}
			if got.Amount != tc.want.Amount || got.Category != tc.want.Category || got.Note != tc.want.Note {
				t.Fatalf("ParseMessage(%q) = %+v, want %+v", tc.text, got, tc.want)
			}
		})
	}
}
