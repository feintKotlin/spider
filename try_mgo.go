package main

import (
	"fmt"
	"log"

	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type Person struct {
	Name  string
	Phone string
}

func main() {
	// connect to mongodb and return a session
	session, err := mgo.Dial("localhost")
	if err != nil {
		log.Printf("Error: %s", err.Error())
	}
	defer session.Close()

	session.SetMode(mgo.Monotonic, true)

	// get a collection object
	c := session.DB("feint").C("people")

	// insert two row to a collection
	err = c.Insert(&Person{"Ale", "+55 53 8116 9639"},
		&Person{"Cla", "+55 53 8402 8510"})

	if err != nil {
		log.Printf("Error: %s", err.Error())
	}

	result := Person{}
	// query a row from current collection
	err = c.Find(bson.M{"name": "Ale"}).One(&result)
	if err != nil {
		log.Printf("Error: %s", err.Error())
	}

	fmt.Println("Phone:", result.Phone)
}
