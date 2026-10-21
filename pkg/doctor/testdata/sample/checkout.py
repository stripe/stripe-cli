"""Runnable local request-shape example. It sends no requests and needs no keys."""

import json
from urllib.request import Request


def build_request(methods):
    params = {"payment_method_types": methods, "mode": "payment"}
    return Request(
        "https://api.stripe.com/v1/checkout/sessions",
        data=json.dumps(params).encode(),
        method="POST",
    )


def main():
    for methods in (["card", "klarna"], ["ideal"]):
        request = build_request(methods)
        print(request.method, request.full_url, request.data.decode())


if __name__ == "__main__":
    main()
