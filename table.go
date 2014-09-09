package main

import "fmt"

// TableSchema holds a channel streaming table information from one of the databases as well as
// a reference to the current row of data we're viewing.
//
// TableSchema implements the Schema interface defined in pgdiff.go
type TableSchema struct {
	channel chan map[string]string
	row     map[string]string
}

// NextRow reads from the channel and tells you if there may be more rows or not
func (c *TableSchema) NextRow(more bool) bool {
	c.row = <-c.channel
	if !more || len(c.row) == 0 {
		return false
	}
	return true
}

// Compare tells you, in one pass, whether or not the first row matches, is less than, or greater than the second row
func (c *TableSchema) Compare(obj interface{}) int {
	c2, ok := obj.(*TableSchema)
	if !ok {
		fmt.Println("Error!!!, Change(...) needs a TableSchema instance", c2)
		return +999
	}

	//fmt.Printf("Comparing %s with %s", c.row["table_name"], c2.row["table_name"])
	val := _compareString(c.row["table_name"], c2.row["table_name"])
	return val
}

// Add returns SQL to add the table
func (c TableSchema) Add() {
	fmt.Printf("CREATE TABLE %s;\n", c.row["table_name"])
}

// Drop returns SQL to drop the table
func (c TableSchema) Drop() {
	fmt.Printf("DROP TABLE IF EXISTS %s;\n", c.row["table_name"])
}

// Change handles the case where the table and column match, but the details do not
func (c TableSchema) Change(obj interface{}) {
	c2, ok := obj.(*TableSchema)
	if !ok {
		fmt.Println("Error!!!, change needs a TableSchema instance", c2)
	}
	//fmt.Printf("Change Table? %s - %s\n", c.row["table_name"], c2.row["table_name"])
}
