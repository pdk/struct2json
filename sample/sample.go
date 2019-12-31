package main

import (
	"go/token"
	"time"
)

// Bar is for testing
type Bar struct {
	x string
	y int
}

// Foo is for testing
type Foo struct {
	a int
	b string `db:"beta"`
	c *int
	d []string `db:"delta" json:"Delta"`
	e map[string]int
	f map[int]Bar
}

// Boink is for testing
type Boink struct {
	Foo
	fset        *token.FileSet
	CreatedAt   time.Time
	UpdatedByID int64
}

func main() {
	println("Hello, World!")
}
