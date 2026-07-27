package main

import "fmt"

func main() {
	connStr := "host=localhost port=5432 user=postgres password=mysecretpassword dbname=postgres sslmode=disable"
	Type := "postgres"

	manager := &DBManager{}

	err := manager.connect(&connStr, &Type)
	if err != nil {
		return
	}
	defer manager.DB.Close()

	err = manager.InitDatabase()
	if err != nil {
		return
	}
	fmt.Print("Success")

}
