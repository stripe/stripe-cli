// legacy Payment Element client — several from-state signals
import {loadStripe} from '@stripe/stripe-js';
import {Elements} from '@stripe/react-stripe-js';
const elements = stripe.elements({clientSecret});
await stripe.confirmCardPayment(clientSecret, {payment_method: {card}});
