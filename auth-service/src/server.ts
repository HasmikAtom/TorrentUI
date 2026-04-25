import { serve } from "@hono/node-server";
import { Hono } from "hono";

const app = new Hono();

app.get("/", (c) => {
  return c.text("Auth Service");
});

const port = process.env.PORT || 3000;

serve({
  fetch: app.fetch,
  port: typeof port === "string" ? parseInt(port, 10) : port,
});
