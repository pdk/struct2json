package main

type Bar struct {
	x string
	y int
}

type Foo struct {
	a int
	b string
	c *int
	d []string
	e map[string]int
	f map[int]Bar
}

func main() {
	println("Hello, World!")
}
