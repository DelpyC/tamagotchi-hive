const map = document.getElementById("map");
const eventsDiv = document.getElementById("events");

// Render everything from JSON
function renderWorld(world) {
    map.innerHTML = "";

    map.style.display = "grid";
    map.style.gridTemplateColumns = `repeat(${world.width}, 30px)`;
    map.style.gridTemplateRows = `repeat(${world.height}, 30px)`;
    map.style.gap = "2px";

    // Render terrain
    for (let y = 0; y < world.height; y++) {
        for (let x = 0; x < world.width; x++) {
            const index = y * world.width + x;
            const cellDiv = document.createElement("div");
            cellDiv.classList.add("cell");

            // Safe check: if terrain exists, use it; else default
            if (world.cells[index] && world.cells[index].terrain) {
                cellDiv.classList.add(world.cells[index].terrain);
            } else {
                cellDiv.classList.add("plains");
            }

            map.appendChild(cellDiv);
        }
    }

    // Render entities
    if (world.entities) {
        world.entities.forEach(e => {
            const index = e.y * world.width + e.x;
            const cell = map.children[index];

            const icon = document.createElement("span");
            icon.textContent = "⬤";
            icon.style.color = e.color;
            icon.style.fontSize = "20px";
            cell.appendChild(icon);
        });
    }

    // Render events
    if (world.events) {
        eventsDiv.innerHTML = "";
        world.events.slice(-5).forEach(ev => {
            const p = document.createElement("p");
            p.textContent = ev;
            eventsDiv.appendChild(p);
        });
    }
}

// Fetch JSON from mock folder
fetch("Test/Test.json")
    .then(res => res.json())
    .then(data => {
        console.log("World loaded:", data);
        renderWorld(data);
    })
    .catch(err => console.error("Failed to load world:", err));
