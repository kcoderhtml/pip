const quips = [
	"Hold on tight, we're revving up the hamster wheel!",
	"Spinning up the server... or at least pretending to.",
	"Initiating launch sequence. Please fasten your seatbelts!",
	"Booting up... because walking was too slow.",
	"Loading server... hope it's not allergic to work today.",
	"Waking up the server. Coffee not included.",
	"Powering up... who left the lights off?",
	"Server starting! Don't worry, I bribed the bugs to behave.",
	"Fueling up with pure chaos energy.",
	"Welcome aboard the S.S. Server! Prepare for adventure.",
	"Firing up the engines... now where's my spark plug?",
	"Initializing... because 'poof' isn't an option.",
	"Starting server... wait, what's my password again?",
	"Time to wake up! Even servers hate Monday mornings.",
	"Deploying... probably not with parachutes.",
	"Starting server. Please hold, optimism loading.",
	"Reviving the server... no paddles needed.",
	"Lighting the fire under this server. Stand back!",
	"Server booting... crossing fingers, toes, and wires.",
	"On your marks, get set... wait, why isn't it going?",
	"Here we go... it's server o'clock!",
	"Please wait while the server decides if it wants to work.",
	"Starting up, because turning off wasn't an option.",
	"Server online! Let's break some records (and hopefully nothing else).",
	"It's alive! The server awakens... and it's probably hungry.",
];

// export default function to log a random quip
export default function quip() {
	const quip = quips[Math.floor(Math.random() * quips.length)];
	return quip;
}
