# tamagotchi-hive
Jeu de simulation de pixel

Formatage du JSON que l'on va utiliser pour le projet pour créer la map:

{
"width": 50,
"height": 50,
"cells": [
//ici il faut définir le type de territoire (ocean, foret, desert, etc...)
{"terrain": "plains"},
]

"entities":[
//Ici l'entité est définit avec sa position sa couleur et ces attributs (peut être dans le futur)
ex: {"type":"tribe","x":2,"y":1,"name":"Francais","population":5,"color":"blue"},
]

"events":[
//ici si événement ajouter au jeu (création de ville, virus qui apparait, etc...)
}]