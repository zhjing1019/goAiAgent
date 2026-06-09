package mysql

import "testing"

func TestSplitDSN(t *testing.T) {
	dbName, serverDSN, err := splitDSN("root:pass@tcp(127.0.0.1:3306)/go_agent?parseTime=true&charset=utf8mb4&loc=Local")
	if err != nil {
		t.Fatal(err)
	}
	if dbName != "go_agent" {
		t.Fatalf("dbName = %q", dbName)
	}
	if serverDSN != "root:pass@tcp(127.0.0.1:3306)/?parseTime=true&charset=utf8mb4&loc=Local" {
		t.Fatalf("serverDSN = %q", serverDSN)
	}
}
