package main

import "fmt"

// ForeignKeySchema holds a channel streaming foreign key data from one of the databases as well as 
// a reference to the current row of data we're viewing.
//
// ForeignKeySchema implements the Schema interface defined in pgdiff.go
type ForeignKeySchema struct {
	channel chan map[string]string
	row     map[string]string
}

// NextRow reads from the channel and tells you if you are at the end or not
func (c *ForeignKeySchema) NextRow(more bool) bool {
	c.row = <-c.channel
	//fmt.Println("Found ", c.row["table_name"])

	if !more || len(c.row) == 0 {
		return false
	}
	return true
}

// Compare tells you, in one pass, whether or not the first row matches, is less than, or greater than the second row
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

// Add returns SQL to add the foreign key
func (c ForeignKeySchema) Add() {
	fmt.Printf("ALTER TABLE %s ADD CONSTRAINT %s FOREIGN KEY(%s) REFERENCES %s(%s);\n", c.row["table_name"], c.row["constraint_name"], c.row["column_name"], c.row["foreign_table_name"], c.row["foreign_column_name"])
}

// Drop returns SQL to drop the foreign key
func (c ForeignKeySchema) Drop() {
	fmt.Printf("ALTER TABLE %s DROP CONSTRAINT IF EXISTS %s;\n", c.row["table_name"], c.row["constraint_name"])
}

// Change handles the case where the table and foreign key name, but the details do not
func (c ForeignKeySchema) Change(obj interface{}) {
	c2, ok := obj.(*ForeignKeySchema)
	if !ok {
		fmt.Println("Error!!!, change needs a ForeignKeySchema instance", c2)
	}
	//fmt.Printf("Change Table? %s - %s\n", c.row["table_name"], c2.row["table_name"])
}
