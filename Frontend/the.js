<!-- JS logic -->

// Sample mock data
const world = {
    width: 50,
    height: 50,
    cells: Array.from({length: 200}, (_, i) => ({
        terrain: i % 3 === 0 ? "plains" : (i % 3 === 1 ? "forest" : "ocean")
    })),
    entities: [
        {type: "tribe", x: 2, y: 1, name: "Hobbits", color: "green"},
        {type: "tribe", x: 5, y: 3, name: "Orcs", color: "red"}
    ],
    events: ["Les Humains ont découverts une nouvelle terre"]
};

const map = document.getElementById("map");
const eventsDiv = document.getElementById("events");

function renderWorld(world) {
    map.innerHTML = "";
    for (let y = 0; y < world.height; y++) {
        for (let x = 0; x < world.width; x++) {
            const cellDiv = document.createElement("div");
            const cell = world.cells[y * world.width + x];
            cellDiv.className = "cell " + cell.terrain;
            map.appendChild(cellDiv);
        }
    }

    // Render entities
    world.entities.forEach(e => {
        const index = e.y * world.width + e.x;
        const cell = map.children[index];
        const icon = document.createElement("span");
        icon.textContent = "🛖"; // small hut for tribe
        icon.style.color = e.color;
        cell.appendChild(icon);
    });

    // Render events
    eventsDiv.innerHTML = "";
    world.events.slice(-5).forEach(ev => {
        const p = document.createElement("p");
        p.textContent = ev;
        eventsDiv.appendChild(p);
    });
}

// Initial render
renderWorld(world);



