# noinspection PyPackageRequirements
import click
import requests
from concurrent.futures import ThreadPoolExecutor
from datetime import datetime
import time


def send_request(worker_id, num_requests, endpoint):
    for i in range(num_requests):
        resp = requests.post(
            url='http://localhost:11888/ws/internal/onsite/trigger',
            json={
                'endpoint': endpoint,
                'data': f'{datetime.now().isoformat()} Sent by {worker_id}, message order {i}'
            }
        )
        if resp.status_code != 200:
            print(f'{worker_id} {resp.json()}')
        time.sleep(0.001)


@click.command()
@click.option('--num-workers', '-w', default=1, help='Number of workers')
@click.option('--num-requests', '-r', default=1, help='Number of requests per worker')
@click.option('--endpoint', '-e', required=True, help='Target endpoint')
def main(num_workers, num_requests, endpoint):
    print(f'num_workers={num_workers}, num_request={num_requests}, endpoint={endpoint}')
    with ThreadPoolExecutor(max_workers=num_workers) as pool:
        for i in range(num_workers):
            pool.submit(send_request, f'worker-{i}', num_requests, endpoint)


if __name__ == '__main__':
    main()
