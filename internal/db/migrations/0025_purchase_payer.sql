-- 0025 · Who is paying, for asking WeChat about a virtual payment order.
--
-- WeChat's /xpay/query_order needs the buyer's openid. It's kept only while the purchase is
-- pending: completing a purchase clears it.

ALTER TABLE credit_purchases ADD COLUMN payer_openid text;
