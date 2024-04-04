package main

type GoFile struct {
	Imports []string `json:"imports"`
	Path    string   `json:"path"`
}
