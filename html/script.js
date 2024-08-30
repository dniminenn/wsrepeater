document.addEventListener("DOMContentLoaded", function () {
    const url = `${window.location.protocol}//${window.location.hostname}:${window.location.port}/latest`;

    function updateWeatherData() {
        fetch(url)
            .then((response) => {
                if (!response.ok) {
                    throw new Error("Network response was not ok");
                }
                return response.json();
            })
            .then((data) => {
                fetch(
                    `${window.location.protocol}//${window.location.hostname}:${window.location.port}/sunrise-sunset`,
                )
                    .then((response) => response.json())
                    .then((sunData) => {
                        const sunrise = new Date(sunData.results.sunrise);
                        const sunset = new Date(sunData.results.sunset);
                        document.getElementById("sunrise").textContent =
                            sunrise.toLocaleTimeString([], {
                                hour: "2-digit",
                                minute: "2-digit",
                            });
                        document.getElementById("sunset").textContent =
                            sunset.toLocaleTimeString([], {
                                hour: "2-digit",
                                minute: "2-digit",
                            });

                        // Determine whether to show sun or moon icon
                        const now = new Date();
                        let iconClass = "";
                        if (now >= sunrise && now < sunset) {
                            iconClass = "fas fa-sun";
                        } else {
                            iconClass = "fas fa-moon";
                        }
                        document.getElementById("temp-icon").className =
                            iconClass;
                    });

                fetch(
                    `${window.location.protocol}//${window.location.hostname}:${window.location.port}/moon`,
                )
                    .then((response) => response.json())
                    .then((moonData) => {
                        document.getElementById("moonPhase").textContent =
                            moonData.phase;
                        document.getElementById(
                            "moonIllumination",
                        ).textContent =
                            `Illumination: ${moonData.illumination.toFixed(1)}%`;

                        const moonIcon = document.getElementById("moon-icon");
                        switch (moonData.phase) {
                            case "New Moon":
                                moonIcon.textContent = "🌑";
                                break;
                            case "Waxing Crescent":
                                moonIcon.textContent = "🌒";
                                break;
                            case "First Quarter":
                                moonIcon.textContent = "🌓";
                                break;
                            case "Waxing Gibbous":
                                moonIcon.textContent = "🌔";
                                break;
                            case "Full Moon":
                                moonIcon.textContent = "🌕";
                                break;
                            case "Waning Gibbous":
                                moonIcon.textContent = "🌖";
                                break;
                            case "Last Quarter":
                                moonIcon.textContent = "🌗";
                                break;
                            case "Waning Crescent":
                                moonIcon.textContent = "🌘";
                                break;
                            default:
                                moonIcon.textContent = "🌑";
                        }
                    });

                const tempC = (((parseFloat(data.tempf) - 32) * 5) / 9).toFixed(
                    1,
                );
                document.getElementById("temp").textContent = tempC;

                const tempInC = (
                    ((parseFloat(data.tempinf) - 32) * 5) /
                    9
                ).toFixed(1);
                document.getElementById("tempIn").textContent = tempInC;
                document.getElementById("uv").textContent = data.uv;

                const humidity = parseFloat(data.humidity);
                document.getElementById("humidity").textContent = humidity;
                const humidityIn = parseFloat(data.humidityin);
                document.getElementById("humidityIn").textContent = humidityIn;
                const windSpeedKmH = (
                    parseFloat(data.windspeedmph) * 1.60934
                ).toFixed(1);
                document.getElementById("windSpeed").textContent = windSpeedKmH;
                const windGustKmH = (
                    parseFloat(data.windgustmph) * 1.60934
                ).toFixed(1);
                document.getElementById("windGust").textContent = windGustKmH;
                const baroHPa = (parseFloat(data.baromrelin) * 33.8639).toFixed(
                    2,
                );
                document.getElementById("baro").textContent = baroHPa;
                const dailyRainMm = (
                    parseFloat(data.dailyrainin) * 25.4
                ).toFixed(2);
                document.getElementById("dailyRain").textContent = dailyRainMm;
                const weeklyRainMm = (
                    parseFloat(data.weeklyrainin) * 25.4
                ).toFixed(2);
                document.getElementById("weeklyRain").textContent =
                    weeklyRainMm;
                const monthlyRainMm = (
                    parseFloat(data.monthlyrainin) * 25.4
                ).toFixed(2);
                document.getElementById("monthlyRain").textContent =
                    monthlyRainMm;
                const rainRateMmH = (
                    parseFloat(data.rainratein) * 25.4
                ).toFixed(2);
                document.getElementById("rainRate").textContent = rainRateMmH;

                // Convert wind direction from degrees to cardinal
                const directions = [
                    "N",
                    "NNE",
                    "NE",
                    "ENE",
                    "E",
                    "ESE",
                    "SE",
                    "SSE",
                    "S",
                    "SSW",
                    "SW",
                    "WSW",
                    "W",
                    "WNW",
                    "NW",
                    "NNW",
                ];
                const index = Math.floor(parseFloat(data.winddir) / 22.5 + 0.5);
                const windDirection = directions[index % 16]; // ensure index wraps correctly
                document.getElementById("windDir").textContent = windDirection;

                const now = new Date();
                const optionsDate = {
                    weekday: "long",
                    year: "numeric",
                    month: "long",
                    day: "numeric",
                };
                const optionsTime = { hour: "2-digit", minute: "2-digit" };

                const formattedDate = now.toLocaleDateString(
                    undefined,
                    optionsDate,
                );
                const formattedTime = now.toLocaleTimeString(
                    undefined,
                    optionsTime,
                );

                const dayTimeElement = document.getElementById("day-time");
                dayTimeElement.innerHTML = `${formattedDate}, ${formattedTime}`;

                let additionalInfo = "";
                if (tempC > 20) {
                    // Calculate dew point
                    const tempFloat = parseFloat(tempC);
                    const dewPointK =
                        (243.04 *
                            (Math.log(humidity / 100) +
                                (17.625 * tempFloat) / (243.04 + tempFloat))) /
                            (17.625 -
                                (Math.log(humidity / 100) +
                                    (17.625 * tempFloat) /
                                        (243.04 + tempFloat))) +
                        273.15;

                    // Calculate humidex
                    const humidex =
                        tempFloat +
                        0.5555 *
                            (6.11 *
                                Math.exp(
                                    5417.753 * (1 / 273.16 - 1 / dewPointK),
                                ) -
                                10);
                    additionalInfo = `<span style="color:#ff5555">Humidex: <span>${humidex.toFixed(1)}</span> °C</span>`;
                } else if (tempC <= 10 && windSpeedKmH > 4.8) {
                    // Calculate wind chill
                    const windChill =
                        13.12 +
                        0.6215 * tempC -
                        11.37 * Math.pow(windSpeedKmH, 0.16) +
                        0.3965 * tempC * Math.pow(windSpeedKmH, 0.16);
                    additionalInfo = `<span style="color:#6272a4">Wind Chill: <span>${windChill.toFixed(1)}</span> °C</span>`;
                }

                if (additionalInfo) {
                    document.getElementById("additional-info").innerHTML =
                        `<strong>${additionalInfo}</strong>`;
                }
            })
            .catch((error) => {
                console.error("Error fetching or processing data:", error);
            });
    }

    updateWeatherData();
    setInterval(updateWeatherData, 30000);

    // Handle opening and closing of modal
    const cards = document.querySelectorAll(".card");
    const modal = document.getElementById("plot-modal");
    const plotContainer = document.getElementById("plot-container");

    // Open modal and plot all data when any card is clicked
    cards.forEach((card) => {
        card.addEventListener("click", function () {
            modal.style.display = "block";
            plotContainer.innerHTML = ""; // Clear previous plots
            fetchAndPlotAll(); // Plot all data
        });
    });

    modal.addEventListener("click", function (event) {
        if (
            !plotContainer.contains(event.target) &&
            !event.target.classList.contains("view-btn")
        ) {
            modal.style.display = "none";
            plotContainer.innerHTML = "";
        }
    });
});
