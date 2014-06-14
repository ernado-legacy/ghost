// setup web-support cookie
window.onload  = function () {
    console.log('detecting support')
    // detect webp support
    if (Modernizr.webp) {
        $.cookie('webp', "1");           
    } else {
        $.cookie('webp', "0");
    }

    // detect html5 audio format support
    if (Modernizr.audio.ogg) {
        $.cookie('audio', "ogg");   
    }

    // priority to aac
    if (Modernizr.audio.aac) {
        $.cookie('audio', "aac");   
    }

    // detect html5 video support
    if (Modernizr.video.h264) {
        $.cookie('video', 'mp4');   
    }

    // priority to webm
    if (Modernizr.video.webm) {
        $.cookie('video', 'webm')
    }
    console.log('support cookies set up')
};

var connection;
const WS_RECONNECT = 1000;

function ws() {
    try {
        var host = window.location.host;
        connection = new WebSocket('ws://' + host + '/realtime');
    } catch(e) {
        console.log("ws error", e)
        return setTimeout( ws, WS_RECONNECT );
    }
    connection.onopen = function () {
        console.log('ws connected')
        connection.send("hello")
    };

    connection.onmessage = function (event) {
        try {
            data = JSON.parse(event.data);
            console.log("got message", data);
        } catch (e) {
            console.log("got", event.data)
        }
    };

    connection.onclose = function () {
        console.log('ws closed; reconnecting')
        return setTimeout( ws, WS_RECONNECT );
    }
}

ws();

var peer = new Peer({host: 'localhost', port: 8081, path: '/'}); 