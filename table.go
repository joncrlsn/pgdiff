package main

import "fmt"

type TableSchema struct {
	channel chan map[string]string
	row     map[string]string
}

// Reads from the channel and converts the end-of-channel value into a boolean
func (c *TableSchema) NextRow(more bool) bool {
	c.row = <-c.channel
    //fmt.Println("Found ", c.row["table_name"])

	if !more || len(c.row) == 0 {
		return false
	}
	return true
}

// Compare
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

// Return SQL to add the table
func (c TableSchema) Add() {
	fmt.Printf("CREATE TABLE %s;\n", c.row["table_name"])
}

// Return SQL to drop the table
func (c TableSchema) Drop() {
    fmt.Printf("DROP TABLE %s;\n", c.row["table_name"])
}

// Handle the case where the table and column match, but the details do not
func (c TableSchema) Change(obj interface{}) {
	c2, ok := obj.(*TableSchema)
	if !ok {
		fmt.Println("Error!!!, change needs a TableSchema instance", c2)
	}
    //fmt.Printf("Change Table? %s - %s\n", c.row["table_name"], c2.row["table_name"])
}
