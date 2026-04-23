class_name PlayerController
extends Node

signal logged_in(user)

const MAX_RETRIES := 3
@export var endpoint: String = "https://example.invalid"
var token := ""

func login(user: String) -> bool:
	token = user
	return true

class Session:
	var active := false

	func close() -> void:
		active = false
