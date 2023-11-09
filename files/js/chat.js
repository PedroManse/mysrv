"use strict";

const id = document.getElementById.bind(document)

let socket = null;
let userId = null;
let userHash = null;

// idea, Package Connect to make use other WS services
function Connect(onmsg) {
	socket = new WebSocket("ws://127.0.0.1:8080/wschat");

	socket.addEventListener("close", (event) => {
		console.log(event);
		alert("WS CLOSED");
		window.location.reload();
	})

	//socket.addEventListener("open", (event) => {
	//	socket.send(userName);
	//});

	socket.addEventListener("message", (event) => {
		if (userId === null) {
			//console.log(`My id is ${event.data}`)
			const info = JSON.parse(event.data);
			userId = info.id
			userHash = info.hash

			socket.send(JSON.stringify({
				action:"set-username",
				id:userId,
				hash:userHash,
				info: {
					email,
					name,
				}
			}))
		} else {
			//console.log("Message from server ", event.data);
			onmsg(JSON.parse(event.data));
		}
	});
}

function onMessage({action, from, msg}) {
	switch (action) {
		case "user-msg":
			id("chatlog").appendChild(createElement(
				"div", {class:"chat-msg"}, [
					createElement("h3", {style: {color: "white"}}, from+":"),
					createElement("h3", {style: {color: "white"}}, msg),
				]
			))
			break;
		case "server-msg":
			id("chatlog").appendChild(createElement(
				"div", {class:"server-msg"}, [
					createElement("p", msg),
				]
			))
			break;
	}
}

window.onload = () => {
	Connect(onMessage)
	//setInterval(()=>{
	//	// TODO: at connect ping server to know time offset
	//	// : for now, -70ms is OK
	//	socket.send(JSON.stringify({
	//		action:"keep-alive",
	//		id: userId,
	//		hash: userHash,
	//	}))
	//}, 15000)

	const sendmsg = ()=>{
		console.log("SENT!", id("msg").value)
		id("chatlog").appendChild(createElement(
			"div", {class:"chat-msg"}, [
				createElement("h3", {style: {color: "white"}}, name+":"),
				createElement("h3", {style: {color: "white"}}, id("msg").value),
			]
		))
		socket.send(JSON.stringify({
			action:"message",
			id:userId,
			hash: userHash,
			info:{msg: id("msg").value},
		}))
	}
	id("send").addEventListener("click", sendmsg)
	id("msg").addEventListener("keydown", ({key})=>{
		if (key === "Enter") {
			sendmsg();
			id("msg").value = "";
		}
	})
}

function createElement(name, elements=[], attributes=null) {
	if (typeof elements === "object" && !Array.isArray(elements)) {
		[elements, attributes] = [attributes, elements];
	}

	const el = document.createElement(name);
	for (const attr in attributes) {
		if (attr === "style") {
			for (const stl in attributes[attr]) {
				el.style[stl] = attributes[attr][stl];
			}
			continue;
		}
		el.setAttribute(attr, attributes[attr]);
	}
	if (Array.isArray(elements)) {
		el.append(...elements);
	} else if (typeof elements === "string") {
		el.innerHTML = elements;
	}
	return el;
}

