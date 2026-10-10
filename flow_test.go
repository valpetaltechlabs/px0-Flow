package main

import "testing"

func TestFlowLayer(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"src/components/AddressForm.tsx", "ui"},
		{"src/pages/checkout.jsx", "ui"},
		{"src/pages/api/upvote.ts", "api"},
		{"src/api/checkout.go", "api"},
		{"src/controllers/orders.ts", "api"},
		{"src/services/checkout.ts", "service"},
		{"src/models/Address.go", "db"},
		{"db/migrations/001_create_users.sql", "db"},
		{"prisma/schema.prisma", "db"},
		{"data/seed.json", "db"},
		{"server.ts", "api"},
		{"lib/checkout.ts", "service"},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := flowLayer(tt.path); got != tt.want {
				t.Fatalf("flowLayer(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestFlowSymbolsForHunks(t *testing.T) {
	symbols := []Symbol{
		{Name: "renderForm", Kind: "func", Line: 2},
		{Name: "submitAddress", Kind: "func", Line: 12},
		{Name: "formatPostalCode", Kind: "func", Line: 27},
	}
	got := flowSymbolsForHunks(symbols, []int{8, 14, 28}, 40)
	if len(got) != 3 || got[0].Name != "renderForm" || got[1].Name != "submitAddress" || got[2].Name != "formatPostalCode" {
		t.Fatalf("flowSymbolsForHunks() = %#v, want all three changed symbols", got)
	}
	got = flowSymbolsForHunks(symbols, []int{20}, 40)
	if len(got) != 1 || got[0].Name != "submitAddress" {
		t.Fatalf("flowSymbolsForHunks() = %#v, want only submitAddress", got)
	}
	if got = flowSymbolsForHunks(symbols, nil, 40); len(got) != 0 {
		t.Fatalf("flowSymbolsForHunks(no marks) = %#v, want none", got)
	}
}
