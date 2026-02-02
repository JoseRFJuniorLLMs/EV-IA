export class VoiceService {
    private ws: WebSocket | null = null;
    private mediaRecorder: MediaRecorder | null = null;
    private audioContext: AudioContext;

    constructor() {
        this.audioContext = new AudioContext();
    }

    async startVoiceSession(token: string): Promise<void> {
        // Conecta ao WebSocket de voz
        this.ws = new WebSocket(`wss://api.sigec-ve.com/ws/voice?token=${token}`);

        this.ws.onopen = () => {
            console.log('Voice session started');
            this.startRecording();
        };

        this.ws.onmessage = async (event) => {
            const response = JSON.parse(event.data);

            // Mostra transcrição
            console.log('AI:', response.text);

            // Reproduz áudio de resposta
            const audioData = Uint8Array.from(atob(response.audio), c => c.charCodeAt(0));
            await this.playAudio(audioData);

            // Atualiza UI com resultado da ação
            if (response.actionResult) {
                this.handleActionResult(response.actionResult);
            }
        };
    }

    private async startRecording(): Promise<void> {
        const stream = await navigator.mediaDevices.getUserMedia({ audio: true });

        this.mediaRecorder = new MediaRecorder(stream, {
            mimeType: 'audio/webm;codecs=opus',
        });

        this.mediaRecorder.ondataavailable = (event) => {
            if (event.data.size > 0 && this.ws?.readyState === WebSocket.OPEN) {
                // Converte para PCM16 e envia
                this.convertAndSend(event.data);
            }
        };

        this.mediaRecorder.start(100); // Chunks de 100ms
    }

    private async convertAndSend(audioBlob: Blob): Promise<void> {
        const arrayBuffer = await audioBlob.arrayBuffer();
        const audioBuffer = await this.audioContext.decodeAudioData(arrayBuffer);

        // Converte para PCM16
        const pcm16 = this.audioBufferToPCM16(audioBuffer);

        // Envia para o backend
        this.ws?.send(pcm16);
    }

    private audioBufferToPCM16(audioBuffer: AudioBuffer): ArrayBuffer {
        const samples = audioBuffer.getChannelData(0);
        const pcm16 = new Int16Array(samples.length);

        for (let i = 0; i < samples.length; i++) {
            const s = Math.max(-1, Math.min(1, samples[i]));
            pcm16[i] = s < 0 ? s * 0x8000 : s * 0x7FFF;
        }

        return pcm16.buffer;
    }

    private async playAudio(audioData: Uint8Array): Promise<void> {
        const audioBuffer = await this.audioContext.decodeAudioData(audioData.buffer);
        const source = this.audioContext.createBufferSource();
        source.buffer = audioBuffer;
        source.connect(this.audioContext.destination);
        source.start();
    }

    stopVoiceSession(): void {
        this.mediaRecorder?.stop();
        this.ws?.close();
    }

    private handleActionResult(result: any) {
        // Implement specific logic for action result
    }
}
