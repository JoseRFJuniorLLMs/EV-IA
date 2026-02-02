import ws from 'k6/ws';
import { check } from 'k6';

export let options = {
    stages: [
        { duration: '2m', target: 100 },  // Ramp up to 100 users
        { duration: '5m', target: 100 },  // Stay at 100
        { duration: '2m', target: 200 },  // Spike to 200
        { duration: '5m', target: 200 },  // Stay at 200
        { duration: '2m', target: 0 },    // Ramp down
    ],
    thresholds: {
        'ws_session_duration': ['p(95)<5000'], // 95% das sessões < 5s
        'checks': ['rate>0.95'],               // 95% success rate
    },
};

export default function () {
    const url = 'wss://api.sigec-ve.com/ws/voice';
    const params = { headers: { 'Authorization': 'Bearer TOKEN' } };

    const res = ws.connect(url, params, function (socket) {
        socket.on('open', () => {
            console.log('Connected');

            // Envia comando de voz simulado
            const audioChunk = new Uint8Array(1024).fill(0);
            socket.send(audioChunk);
        });

        socket.on('message', (data) => {
            const response = JSON.parse(data);
            check(response, {
                'has text': (r) => r.text !== undefined,
                'has audio': (r) => r.audio !== undefined,
                'has intent': (r) => r.intent !== undefined,
            });
            socket.close();
        });

        socket.setTimeout(() => {
            socket.close();
        }, 5000);
    });

    check(res, { 'status is 101': (r) => r && r.status === 101 });
}
