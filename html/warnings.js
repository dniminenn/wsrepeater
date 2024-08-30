document.addEventListener("DOMContentLoaded", function () {
    const kentCountyUrl = `${window.location.protocol}//${window.location.hostname}:${window.location.port}/rss/nb10_e.xml`;
    const westmorlandCountyUrl = `${window.location.protocol}//${window.location.hostname}:${window.location.port}/rss/nb16_e.xml`;

    const kentBadge = document.getElementById("kentBadge");
    const westmorlandBadge = document.getElementById("westmorlandBadge");

    function fetchWarning(url, badgeElement) {
        fetch(url)
            .then((response) => response.text())
            .then((str) =>
                new window.DOMParser().parseFromString(str, "text/xml"),
            )
            .then((data) => {
                const entries = data.querySelectorAll("entry summary");
                let colorClass = "green";
                let count = 0;

                entries.forEach((entry) => {
                    const summary = entry.textContent.toLowerCase();

                    if (summary.includes("warning")) {
                        colorClass = "red";
                        count++;
                    } else if (summary.includes("watch")) {
                        if (colorClass !== "red") colorClass = "yellow";
                        count++;
                    } else if (summary.includes("statement")) {
                        if (colorClass !== "red" && colorClass !== "yellow")
                            colorClass = "grey";
                        count++;
                    }
                });

                badgeElement.className = `warning-badge ${colorClass}`;
                if (count > 0) {
                    badgeElement.innerHTML =
                        badgeElement.textContent + `<span>${count}</span>`;
                } else {
                    badgeElement.innerHTML = badgeElement.textContent;
                }
            })
            .catch((error) => {
                console.error("Error fetching or processing data:", error);
            });
    }

    fetchWarning(kentCountyUrl, kentBadge);
    fetchWarning(westmorlandCountyUrl, westmorlandBadge);

    // Update every 10 minutes
    setInterval(() => {
        fetchWarning(kentCountyUrl, kentBadge);
        fetchWarning(westmorlandCountyUrl, westmorlandBadge);
    }, 600000);
});
