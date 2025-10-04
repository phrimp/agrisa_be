import z from 'zod';

const pingReponseSchema = z.object({
  message: z.string().default('pong'),
});

type PingResponse = z.infer<typeof pingReponseSchema>;

export { pingReponseSchema };
export type { PingResponse };
