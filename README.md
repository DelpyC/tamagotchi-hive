# Tamagotchi Hive

## Présentation

Tamagotchi Hive est un projet de simulation stratégique développé en Go. Il met en scène plusieurs civilisations contrôlées par une intelligence artificielle évoluant de manière autonome sur une carte générée procéduralement.

L'objectif du projet est de simuler le développement de civilisations capables de gérer leurs ressources, de fonder des villes, d'effectuer des recherches technologiques et de construire des infrastructures afin d'assurer leur croissance.

Le projet s'inspire des mécaniques de jeux de stratégie en temps réel et de gestion, tout en mettant l'accent sur l'autonomie des agents et les interactions entre les différents systèmes du jeu.

---

## Fonctionnalités

### Génération du monde

* Génération procédurale de la carte.
* Différents types de terrains.
* Placement des ressources naturelles.
* Création d'un environnement de jeu dynamique.

### Civilisations autonomes

Chaque civilisation est capable de :

* Explorer son environnement.
* Collecter des ressources.
* Fonder des villes.
* Développer ses infrastructures.
* Effectuer des recherches technologiques.
* Construire des bâtiments et des merveilles.

### Gestion des villes

Les villes constituent le cœur du développement des civilisations et permettent :

* La production de ressources.
* Le développement démographique.
* La construction de bâtiments.
* L'amélioration de l'économie locale.

### Arbre technologique

Le système de recherche permet aux civilisations de débloquer de nouvelles améliorations et d'accéder à des bâtiments ou constructions plus avancés.

### Gestion des ressources

Le projet intègre plusieurs types de ressources nécessaires au développement des civilisations, notamment les ressources alimentaires, économiques et stratégiques.

### Interface utilisateur

Une interface web permet de visualiser en temps réel l'évolution de la simulation et les différents éléments du monde généré.

---

## Technologies utilisées

### Backend

* Go

### Frontend

* HTML
* CSS
* JavaScript

### Communication

* JSON
* WebSocket

### Gestion de projet

* Git
* GitHub

---

## Architecture du projet

Le projet est organisé selon une architecture séparant les différents composants du système :

```text
tamagotchi-hive/
│
├── cmd/
├── internal/
├── web/
├── assets/
├── data/
└── README.md
```

Cette organisation permet de distinguer la logique métier, les données et l'interface utilisateur.

---

## Installation

### Cloner le dépôt

```bash
git clone https://github.com/DelpyC/tamagotchi-hive.git
```

### Accéder au projet

```bash
cd tamagotchi-hive
```

### Installer les dépendances

```bash
go mod tidy
```

### Lancer le projet

```bash
go run .
```

Selon l'organisation du projet :

```bash
go run cmd/main.go
```

---

## Utilisation

1. Lancer le serveur.
2. Ouvrir l'application web.
3. Observer le développement autonome des civilisations.
4. Suivre leur évolution et leurs interactions au cours de la simulation.

---

## Objectifs pédagogiques

Ce projet a permis de mettre en pratique plusieurs notions importantes :

* Programmation en Go.
* Conception d'architectures logicielles.
* Développement d'algorithmes de simulation.
* Communication en temps réel avec WebSocket.
* Développement d'une interface web.
* Gestion de projet avec Git et GitHub.
* Travail collaboratif.

---

## Perspectives d'amélioration

Les évolutions envisagées comprennent notamment :

* Un système diplomatique entre les civilisations.
* L'ajout de mécanismes de combat.
* L'amélioration des comportements des intelligences artificielles.
* L'enrichissement de l'interface utilisateur.
* La sauvegarde et le chargement des parties.
* L'ajout de statistiques et d'outils d'analyse de la simulation.

---

## Auteurs

Projet réalisé dans le cadre d'un projet académique par KING Dylan.

---

## Licence

Ce projet a été développé dans un cadre pédagogique.
