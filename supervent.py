import json
import random
import time
from datetime import datetime, timezone
import uuid
import asyncio
import aiohttp
import math
import yaml
import argparse
import signal
import sys

DEFAULT_BATCH_SIZE = 100

class EventGenerator:
    def __init__(self, dataset, api_key, batch_size=DEFAULT_BATCH_SIZE):
        self.dataset = dataset
        self.api_key = api_key
        self.url = f"https://api.axiom.co/v1/datasets/{dataset}/ingest"
        self.batch_size = batch_size
        self.batch = []

    async def emit(self, record):
        # Strip "custom." prefix from keys
        stripped_record = {k.replace("custom_", ""): v for k, v in record.items()}
        # Add a timestamp to the record
        stripped_record['_time'] = datetime.now(timezone.utc).isoformat()

        self.batch.append(stripped_record)
        if len(self.batch) >= self.batch_size:
            await self.send_batch()

    async def send_batch(self):
        if not self.batch:
            return
        # print("sending batch")
        headers = {
            "Content-Type": "application/json",
            "Authorization": f"Bearer {self.api_key}"
        }
        async with aiohttp.ClientSession() as session:
            async with session.post(self.url, headers=headers, json=self.batch) as response:
                if response.status != 200:
                    print(f"Failed to send batch: {response.status}")
            # else:
                # print("Batch sent successfully")
        self.batch = []

# Load configuration from config.json
def load_config(file_path):
    with open(file_path, 'r') as file:
        config = json.load(file)
    return config

# Load AXIOM_DATASET, AXIOM_API_KEY, and HTTP_BATCH_SIZE from axiom_config.yaml
def load_axiom_config(file_path):
    with open(file_path, 'r') as file:
        config = yaml.safe_load(file)
    return config

# Generate a random event based on the source configuration
def generate_event(source_config):
    event = {"source": source_config["vendor"]}
    for field, details in source_config['fields'].items():
        if details['type'] == 'datetime':
            if source_config['timestamp_format'] == 'UTC':
                event[field] = datetime.utcnow().strftime(details.get('format', '%Y-%m-%dT%H:%M:%SZ'))
            elif source_config['timestamp_format'] == 'ISO':
                event[field] = datetime.now().isoformat()
            elif source_config['timestamp_format'] == 'Unix':
                event[field] = int(time.time())
            elif source_config['timestamp_format'] == 'RFC3339':
                event[field] = datetime.now().strftime('%Y-%m-%dT%H:%M:%SZ')
            else:
                event[field] = datetime.now().strftime(source_config['timestamp_format'])
        elif details['type'] == 'string':
            if 'constraints' in details and 'allowed_values' in details['constraints']:
                event[field] = random.choice(details['constraints']['allowed_values'])
            else:
                if field == "message":
                    if 'formats' in details:
                        selected_format = random.choice(details['formats'])
                        event[field] = selected_format.format(
                            timestamp=datetime.now().strftime(details.get('format', '%Y-%m-%dT%H:%M:%SZ')),
                            src_ip=generate_random_ip_address() if '{src_ip}' in selected_format else '',
                            dst_ip=generate_random_ip_address() if '{dst_ip}' in selected_format else '',
                            ip_address=generate_random_ip_address() if '{ip_address}' in selected_format else ''
                        )
                    else:
                        event[field] = details['format'].format(
                            timestamp=datetime.now().strftime(details.get('format', '%Y-%m-%dT%H:%M:%SZ')),
                            src_ip=generate_random_ip_address() if '{src_ip}' in details['format'] else '',
                            dst_ip=generate_random_ip_address() if '{dst_ip}' in details['format'] else '',
                            ip_address=generate_random_ip_address() if '{ip_address}' in details['format'] else ''
                        )
                else:
                    event[field] = generate_random_username() if field == "user" else generate_random_ip_address()
        elif details['type'] == 'int':
            min_val = int(details.get('constraints', {}).get('min', 0))
            max_val = int(details.get('constraints', {}).get('max', 100))
            event[field] = random.randint(min_val, max_val)
        # Add more types as needed
    return event

# Generate a random IP address
def generate_random_ip_address():
    return f"{random.randint(1, 255)}.{random.randint(0, 255)}.{random.randint(0, 255)}.{random.randint(1, 255)}"

# Generate a random username based on Zipf's law
def generate_random_username():
    usernames = [
        "john_doe", "jane_smith", "mohamed_ali", "li_wei", "maria_garcia",
        "yuki_tanaka", "olga_petrov", "raj_kumar", "fatima_zahra", "chen_wang",
        "ahmed_hassan", "isabella_rossi", "david_jones", "sophia_martinez", "emily_clark",
        "noah_brown", "mia_wilson", "lucas_miller", "oliver_davis", "ava_moore",
        "ethan_taylor", "amelia_anderson", "james_thomas", "harper_jackson", "benjamin_white",
        "liam_johnson", "emma_rodriguez", "william_lee", "sophia_kim", "mason_martin",
        "elijah_hernandez", "logan_lopez", "alexander_gonzalez", "sebastian_perez", "daniel_hall",
        "matthew_young", "henry_king", "jack_wright", "levi_scott", "isaac_green",
        "gabriel_baker", "julian_adams", "jayden_nelson", "lucas_carter", "anthony_mitchell",
        "grayson_perez", "dylan_roberts", "leo_turner", "jaxon_phillips", "asher_campbell",
        "ananya_sharma", "arjun_patel", "priya_singh", "vikram_gupta", "neha_verma",
        "sanjay_rana", "deepika_kapoor", "ravi_mehta", "sara_khan", "manoj_joshi",
        "željko_ivanović", "šime_šarić", "đorđe_đorđević", "čedomir_čolić", "žana_živković",
        "miloš_milošević", "ana_marija", "ivan_ivanov", "petar_petrov", "nikola_nikolić",
        "marta_novak", "katarina_kovač", "tomaž_tomažič", "matej_matejić", "vanja_vuković",
        "dragana_dimitrijević", "bojan_bojović", "milica_milovanović", "stefan_stefanović", "vanja_vasić",
        "igor_ilić", "jelena_jovanović", "marko_marković", "tanja_tomić", "zoran_zorić"
    ]

    # Generate weights using Zipf's law
    weights = [1.0 / (i + 1) for i in range(len(usernames))]
    total_weight = sum(weights)
    normalized_weights = [w / total_weight for w in weights]

    # Select a username based on the weights
    r = random.random()
    cumulative_weight = 0.0
    for username, weight in zip(usernames, normalized_weights):
        cumulative_weight += weight
        if r < cumulative_weight:
            return username
    return usernames[-1]

# Signal handler to gracefully exit on ^C or kill signal
def signal_handler(signal, frame):
    print("Received interrupt signal, sending remaining events...")
    if len(event_generator.batch) > 0:
        asyncio.create_task(event_generator.send_batch())
    sys.exit(0)

# Main function to generate events
async def main():
    parser = argparse.ArgumentParser(description='Generate and send events.')
    parser.add_argument('--config', type=str, default='config.json', help='Path to the configuration file')
    args = parser.parse_args()

    config = load_config(args.config)
    axiom_config = load_axiom_config('axiom_config.yaml')
    dataset = axiom_config['AXIOM_DATASET']
    api_key = axiom_config['AXIOM_API_KEY']
    batch_size = int(axiom_config.get('HTTP_BATCH_SIZE', DEFAULT_BATCH_SIZE))
    global event_generator
    event_generator = EventGenerator(dataset=dataset, api_key=api_key, batch_size=batch_size)

    # Set up signal handling to gracefully exit on ^C or kill signal
    signal.signal(signal.SIGINT, signal_handler)
    signal.signal(signal.SIGTERM, signal_handler)

    # Generate events in a round-robin fashion indefinitely
    while True:
        for source in config['sources']:
            event = generate_event(source)
            await event_generator.emit(event)

if __name__ == "__main__":
    asyncio.run(main())