#!/bin/bash

IP="34.68.149.84"  # Replace with the actual IP address
PORT="8081"  # Replace with the actual port number

# Array of 10 different questions
questions=(
  "Write as if you were a critic: Waterloo Canada"
  "What are the pros and cons of living in Waterloo Canada?"
  "Is Waterloo Canada a good place to raise a family?"
  "What are the best things to do in Waterloo Canada?"
  "What is the cost of living in Waterloo Canada?"
  "How is the job market in Waterloo Canada?"
  "What are the best schools in Waterloo Canada?"
  "What is the weather like in Waterloo Canada?"
  "What is the culture like in Waterloo Canada?"
  "What are some fun facts about Waterloo Canada?"
)

while true; do
  # Generate a random index between 0 and 9
  random_index=$((RANDOM % 10))

  # Get the question at the random index
  question="${questions[$random_index]}"

  curl -i ${IP}:${PORT}/v1/completions \
    -H 'Content-Type: application/json' \
    -d "{
      \"model\": \"tweet-summary\",
      \"prompt\": \"$question\",
      \"max_tokens\": 100,
      \"temperature\": 0
    }"

  sleep 10
done

