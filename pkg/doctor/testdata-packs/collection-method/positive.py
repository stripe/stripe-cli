import stripe

# Invoice create — top-level billing
stripe.Invoice.create(
    customer="cus_x",
    billing="send_invoice",
)

# Invoice update (SDK convention: .modify) — top-level billing
stripe.Invoice.modify(
    "in_x",
    billing="send_invoice",
)
