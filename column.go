package main

import "fmt"

type ColumnSchema struct {
	channel chan map[string]string
	row     map[string]string
}

// Reads from the channel and converts the end-of-channel value into a boolean
func (c *ColumnSchema) NextRow(more bool) bool {
	c.row = <-c.channel
	if !more || len(c.row) == 0 {
       return false
	}
	return true
}

// Compare
func (c ColumnSchema) Compare(obj interface{}) int {
	c2, ok := obj.(*ColumnSchema)
	if !ok {
		fmt.Println("Error!!!, change needs a ColumnSchema instance", c2)
	}

	val := _compareString(c.row["table_name"], c2.row["table_name"])
	if val != 0 {
		// Table name differed so return that value
		return val
	}

	// Table name was the same so compare column name
	val = _compareString(c.row["column_name"], c2.row["column_name"])
	return val
}

// Return SQL to add the column
func (c ColumnSchema) Add() {
	if c.row["data_type"] == "character varying" {
		fmt.Printf("ALTER TABLE %s ADD COLUMN %s %s(%s)", c.row["table_name"], c.row["column_name"], c.row["data_type"], c.row["character_maximum_length"])
	} else {
		fmt.Printf("ALTER TABLE %s ADD COLUMN %s %s", c.row["table_name"], c.row["column_name"], c.row["data_type"])
	}

	if c.row["is_nullable"] == "NO" {
		fmt.Printf(" NOT NULL")
	}
	if c.row["column_default"] != "null" {
		fmt.Printf(" DEFAULT %s", c.row["column_default"])
	}
	fmt.Printf(";\n")
}

// Return SQL to drop the column
func (c ColumnSchema) Drop() {
	// if dropping column
    fmt.Printf("ALTER TABLE %s DROP COLUMN %s;\n", c.row["table_name"], c.row["column_name"])
}

// Handle the case where the table and column match, but the details do not
func (c ColumnSchema) Change(obj interface{}) {
	c2, ok := obj.(*ColumnSchema)
	if !ok {
		fmt.Println("Error!!!, change needs a ColumnSchema instance", c2)
	}
//    fmt.Printf("Changes? ")
//	// if changing type
//	if c.row["data_type"] == "character varying" {
//		// varchar needs a length specified
//		fmt.Printf("ALTER TABLE %s ALTER COLUMN %s TYPE %s(%s);\n", c.row["table_name"], c.row["column_name"], c.row["data_type"], c.row["character_maximum_length"])
//	} else {
//		fmt.Printf("ALTER TABLE %s ALTER COLUMN %s TYPE %s;\n", c.row["table_name"], c.row["column_name"], c.row["data_type"])
//	}
//
//	// if changing/adding default value
//	fmt.Printf("ALTER TABLE %s ALTER COLUMN %s SET DEFAULT %s;\n", c.row["table_name"], c.row["column_name"], c.row["column_default"])
//
//	// if dropping default value
//	fmt.Printf("ALTER TABLE %s ALTER COLUMN %s DROP DEFAULT;\n", c.row["table_name"], c.row["column_name"])
//
//	// if adding not null
//	fmt.Printf("ALTER TABLE %s ALTER COLUMN %s SET NOT NULL;\n", c.row["table_name"], c.row["column_name"])
//
//	// if dropping not null
//	fmt.Printf("ALTER TABLE %s ALTER COLUMN %s DROP NOT NULL;\n", c.row["table_name"], c.row["column_name"])
//	return "Change"
}
