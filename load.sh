#!/bin/bash

IP="34.68.149.84"  # Replace with the actual IP address
PORT="8081"  # Replace with the actual port number

# Array of 10 different questions
questions=(
  "What's the capital of France?"
  "Explain the theory of relativity in simple terms."
  "How many bones are in the human body?"
  "Which is better, a cat or a dog, and why?"
  "Compare and contrast the music of Beethoven and Mozart."
  "Write a haiku about a rainy day."
  "If you could have any superpower, what would it be and why?"
  "What are some hidden gems or unique attractions in Waterloo, Canada?"
  "How does the tech industry in Waterloo compare to that in Toronto?"
  "What are the current top trending topics on Twitter?"
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
      \"max_tokens\": 200,
      \"temperature\": 0
    }"

  sleep 10
done

