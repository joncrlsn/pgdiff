package main

import "fmt"

type ForeignKeySchema struct {
	channel chan map[string]string
	row     map[string]string
}

// Reads from the channel and converts the end-of-channel value into a boolean
func (c *ForeignKeySchema) NextRow(more bool) bool {
	c.row = <-c.channel
    //fmt.Println("Found ", c.row["table_name"])

	if !more || len(c.row) == 0 {
		return false
	}
	return true
}

// Compare
func (c *ForeignKeySchema) Compare(obj interface{}) int {
	c2, ok := obj.(*ForeignKeySchema)
	if !ok {
		fmt.Println("Error!!!, Change(...) needs a ForeignKeySchema instance", c2)
        return +999
	}

	//fmt.Printf("Comparing %s with %s", c.row["table_name"], c2.row["table_name"])
	val := _compareString(c.row["table_name"], c2.row["table_name"])
    if val != 0 {
        return val
    }

	val = _compareString(c.row["constraint_name"], c2.row["constraint_name"])
	return val
}

// Return SQL to add the table
func (c ForeignKeySchema) Add() {
	fmt.Printf("ALTER TABLE %s ADD CONSTRAINT %s FOREIGN KEY(%s) REFERENCES %s(%s);\n", c.row["table_name"], c.row["constraint_name"], c.row["column_name"], c.row["foreign_table_name"], c.row["foreign_column_name"], )
}

// Return SQL to drop the table
func (c ForeignKeySchema) Drop() {
    fmt.Printf("ALTER TABLE %s DROP CONSTRAINT %s;\n", c.row["table_name"], c.row["constraint_name"])
}

// Handle the case where the table and column match, but the details do not
func (c ForeignKeySchema) Change(obj interface{}) {
	c2, ok := obj.(*ForeignKeySchema)
	if !ok {
		fmt.Println("Error!!!, change needs a ForeignKeySchema instance", c2)
	}
    //fmt.Printf("Change Table? %s - %s\n", c.row["table_name"], c2.row["table_name"])
}
