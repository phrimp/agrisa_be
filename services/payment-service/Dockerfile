# Use an official Bun runtime as the base image
FROM oven/bun:latest

# Set the working directory in the container
WORKDIR /app

# Copy package.json and package-lock.json (if available) from the correct location
COPY services/payment-service/package*.json ./

# Install dependencies using Bun
RUN bun install

# Copy the rest of the application code
COPY ./services/payment-service .

# Define the command to run the app using Bun
CMD ["bun", "run", "start"]