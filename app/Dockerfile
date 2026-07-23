FROM node:22-alpine AS build
WORKDIR /src
RUN corepack enable
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
RUN pnpm install --frozen-lockfile
COPY . .
RUN pnpm build

FROM node:22-alpine
WORKDIR /app
COPY --from=build /src/.output .output
EXPOSE 3000
ENV HOST=0.0.0.0 PORT=3000
CMD ["node", ".output/server/index.mjs"]
