# Heading 1

## Heading 2

### Heading 3

#### Heading 4

##### Heading 5

###### Heading 6

---

## Paragraph & Inline Styles

This is a regular paragraph with **bold text**, *italic text*, and ***bold italic text***.
You can also use ~~strikethrough~~ for deleted content.
Inline `code` appears within a sentence like this.

---

## Blockquote

> This is a blockquote. It can span multiple lines and contain other elements.
>
> It can have multiple paragraphs, and even **bold** or *italic* text inside.
>
> > Nested blockquotes are also supported.

---

## Unordered List

- Item one
- Item two
  - Nested item A
  - Nested item B
    - Deeply nested item
- Item three

---

## Ordered List

1. First item
2. Second item
   1. Nested ordered item
   2. Another nested item
3. Third item

---

## Task List

- [x] Write the markdown showcase
- [x] Add headings and paragraphs
- [ ] Add more block types
- [ ] Review styles

---

## Code Block (Go)

```go
package main

import "fmt"

func main() {
    fmt.Println("Hello, world!")
}
```

## Code Block (Shell)

```sh
stripe docs search "payment intents"
```

## Code Block (JSON)

```json
{
  "id": "pi_123",
  "object": "payment_intent",
  "amount": 2000,
  "currency": "usd",
  "status": "succeeded"
}
```

## Code Block (no language)

```
plain text block
no syntax highlighting
```

---

## Horizontal Rule

Above the rule.

---

Below the rule.

---

## Links & Images

A [link to docs.stripe.com](https://docs.stripe.com) in a sentence.

An image:

![Stripe logo](https://stripe.com/img/v3/home/twitter.png)

---

## Table

| Name        | Type    | Description                     |
|-------------|---------|----------------------------------|
| `id`        | string  | Unique identifier for the object |
| `amount`    | integer | Amount in the smallest currency unit |
| `currency`  | string  | Three-letter ISO currency code   |
| `status`    | string  | Current status of the payment    |

---

## Definition List

Term one
: Definition of term one.

Term two
: First definition of term two.
: Second definition of term two.
