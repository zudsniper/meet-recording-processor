#!/usr/bin/env python3
import argparse
import json
import sys
from time import perf_counter

def main():
    p = argparse.ArgumentParser()
    p.add_argument('--audio', required=True)
    p.add_argument('--model', default='base.en')
    p.add_argument('--device', default='auto')  # auto|cpu|cuda
    args = p.parse_args()

    try:
        from faster_whisper import WhisperModel
    except Exception as e:
        sys.stderr.write('faster-whisper not installed. pip install faster-whisper\n')
        sys.exit(2)

    device = None
    compute_type = None
    if args.device == 'auto':
        device = 'auto'
    elif args.device == 'cuda':
        device = 'cuda'
        compute_type = 'float16'
    else:
        device = 'cpu'
        compute_type = 'int8'

    t0 = perf_counter()
    try:
        model = WhisperModel(args.model, device=device if device!='auto' else None, compute_type=compute_type)
        device_used = device
    except Exception as e:
        # Fallback to CPU if CUDA/cuDNN missing or broken
        msg = str(e).lower()
        if args.device == 'cuda' and ("cudnn" in msg or "cuda" in msg or "invalid handle" in msg):
            sys.stderr.write('CUDA initialization failed; falling back to CPU (int8).\n')
            device_used = 'cpu'
            model = WhisperModel(args.model, device=None, compute_type='int8')
        else:
            raise
    segments, info = model.transcribe(args.audio)
    out = {
        'language': getattr(info, 'language', ''),
        'duration': getattr(info, 'duration', 0.0),
        'device_used': device_used,
        'segments': [
            {
                'start': float(s.start),
                'end': float(s.end),
                'text': s.text.strip(),
            }
            for s in segments
        ],
    }
    sys.stdout.write(json.dumps(out))

if __name__ == '__main__':
    main()
