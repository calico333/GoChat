$(function () {
    let websocket = new WebSocket("wss://" + window.location.host + "/websocket");
    let room = $("#chat-text");
    let msgNum = 0
    let lastUsr = ""

    websocket.addEventListener("message", function (e) { // Handing incoming msg.
        let data = JSON.parse(e.data);
        let chatContent;

        if (lastUsr == data.username) {
            chatContent = `<p id="` + msgNum.toString() + `"></p>`;
            room.append(chatContent);
        } else {
            chatContent = `<hr/><p><strong id="` + msgNum.toString() + "s" +`"></strong></p><p id="` + msgNum.toString() + `"></p>`;
            room.append(chatContent);
            document.getElementById(msgNum.toString() + "s").innerText = data.username;
        }

        document.getElementById(msgNum.toString()).innerText = data.text;

        // +${data.username}
        // : ${data.text}

        //document.getElementById(msgNum.toString()).innerText = data.text

        room.scrollTop = room.scrollHeight; // Auto scroll to the bottom.

        msgNum++;
        lastUsr = data.username;
    });

    $("#input-form").on("submit", function (event) { // Sending outgoing msg.
        event.preventDefault();
        let username = $("#input-username")[0].value;
        let text = $("#input-text")[0].value;

        websocket.send( // Sending json data
            JSON.stringify({
                username: username,
                text: text,
            })
        );

        $("#input-text")[0].value = ""; // Clear the input field.
    });
});