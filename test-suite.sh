#!/bin/bash

server="http://localhost:9001"
opts="--header 'Content-Type: application/json' --silent"

now=$(date +%s%3N)

email="${now}@example.com"
upper_cased_email="${now}@EXAMPLE.com"
bad_email="x@x"

password="open1234"
bad_password="open123"

# ============================
# POST /signups
# ============================
signup_cmd="curl $opts --request POST --data '{\"email\": \"${email}\", \"password\": \"${password}\"}' $server/signup"
printf "Signup: $signup_cmd\n"

signup_res=$(eval $signup_cmd)
echo $signup_res

if [[ "$signup_res" != *"\"id\":"* ]]; then
  printf "FAIL: Should return a new user id on successful signup\n"
  exit 1
fi

match_number="\"id\":[0-9]+"

if [[ ! "$signup_res" =~ $match_number ]]; then
  printf "FAIL: Should return a new user id as a number on successful signup\n"
  exit 1
fi

user_id=$(echo $signup_res | jq --raw-output '.id')
printf "\nCreated new user: ${email} (id: ${user_id})\n"

# Should not allow signup with week password
signup_cmd="curl $opts --request POST --data '{\"email\": \"${email}\", \"password\": \"${bad_password}\"}' $server/signup"
printf "Signup: $signup_cmd\n"

signup_res=$(eval $signup_cmd)
echo $signup_res

if [[ "$signup_res" != *"\"error\":"* ]]; then
  printf "FAIL: Should not allow signup with week password\n"
  exit 1
fi

# Should not allow signup with invalid email
signup_cmd="curl $opts --request POST --data '{\"email\": \"${bad_email}\", \"password\": \"${password}\"}' $server/signup"
printf "Signup: $signup_cmd\n"

signup_res=$(eval $signup_cmd)
echo $signup_res

if [[ "$signup_res" != *"\"error\":"* ]]; then
  printf "FAIL: Should not allow signup with invalid email\n"
  exit 1
fi

# Should not allow signup for already signed up user
signup_res=$(eval $signup_cmd)
echo $signup_res

if [[ "$signup_res" != *"\"error\":"* ]]; then
  printf "FAIL: Should not allow signup for already signed up user\n"
  exit 1
fi

# ============================
# POST /login
# ============================
login_cmd="curl $opts --request POST --data '{\"email\": \"${email}\", \"password\": \"${password}\"}' $server/login"
printf "Login: $login_cmd\n"

login_res=$(eval $login_cmd)
echo $login_res

if [[ "$login_res" != *"\"token\":"* ]]; then
  printf "FAIL: Should return a token on successful login\n"
  exit 1
fi

# Should not allow login with invalid credentials
login_cmd="curl $opts --request POST --data '{\"email\": \"wrong${email}\", \"password\": \"${password}\"}' $server/login"
printf "Login: $login_cmd\n"

login_res=$(eval $login_cmd)
echo $login_res

if [[ "$login_res" == *"\"token\":"* ]]; then
  printf "FAIL: Should not return a token on unsuccessful login\n"
  exit 1
fi

# Email in credential should be case insensitive
login_cmd="curl $opts --request POST --data '{\"email\": \"${upper_cased_email}\", \"password\": \"${password}\"}' $server/login"
printf "Login: $login_cmd\n"

login_res=$(eval $login_cmd)
echo $login_res

if [[ "$login_res" != *"\"token\":"* ]]; then
  printf "FAIL: Email in credential should be case insensitive\n"
  exit 1
fi

token=$(echo $login_res | jq --raw-output '.token')
printf "\nGot token: ${token}\n"

auth_header="--header 'Authorization: Bearer ${token}'"

# ============================
# GET /secret
# ============================
secret_cmd="curl $opts --request GET $auth_header $server/secret"
printf "Secret: $secret_cmd\n"

secret_res=$(eval $secret_cmd)
echo $secret_res

if [[ "$secret_res" != *"\"user_id\":"* ]]; then
  printf "FAIL: Should return user_id together with secret string\n"
  exit 1
fi

if [[ "$secret_res" != *"\"secret\":\"All your base are"* ]]; then
  printf "FAIL: Should return correct secret string\n"
  exit 1
fi

user_id_from_secret=$(echo $secret_res | jq --raw-output '.user_id')
secret=$(echo $secret_res | jq --raw-output '.secret')
printf "\nGot secret  : ${secret}\n"
printf "Got user id : ${user_id_from_secret}\n\n"

if [[ "$user_id" != "$user_id_from_secret" ]]; then
  printf "FAIL: Should get token owner's user_id returned from /secret, expected $user_id but got ${user_id_from_secret}\n"
  exit 1
fi

# ============================
# PATCH /signup/{id} - email
# ============================
patch_email_cmd="curl $opts --request PATCH --data '{\"email\": \"updated-${email}\"}' $auth_header $server/users/${user_id}"
printf "Update: $patch_email_cmd\n"

patch_email_res=$(eval $patch_email_cmd)
echo $patch_email_res

if [[ "$patch_email_res" != *"\"email\":\"updated-${email}"* ]]; then
  printf "FAIL: Should return user with updated email address\n"
  exit 1
fi

if [[ "$patch_email_res" != *"\"updated_at\":\"20"* ]]; then
  printf "FAIL: Should return a valid date in the updated_at field\n"
  exit 1
fi

updated_email=$(echo $patch_email_res | jq --raw-output '.email')

printf "\nUpdated user's email address from: ${email} -> ${updated_email}\n\n"

# =========================================
# FAIL PATCH /users/{id} with invalid email
# =========================================
fail_patch_email_cmd="curl $opts --request PATCH --data '{\"email\": \"${bad_email}\"}' $auth_header $server/users/${user_id}"
printf "Update with bad email: $fail_patch_email_cmd\n"

fail_patch_email_res=$(eval $fail_patch_email_cmd)
echo $fail_patch_email_res

if [[ "$fail_patch_email_res" != *"\"error\":\"validation error: email\""* ]]; then
  printf "FAIL: Should return a validation error for email field\n"
  exit 1
fi

printf "\n"

# ============================
# PATCH /users/{id} - password
# ============================
patch_password_cmd="curl $opts --request PATCH --data '{\"password\": \"1234open\"}' $auth_header $server/users/${user_id}"
printf "Update: $patch_password_cmd\n"

patch_password_res=$(eval $patch_password_cmd)
echo $patch_password_res

if [[ "$patch_password_res" == *"\"password\":\""* ]]; then
  printf "FAIL: Should not return user's password\n"
  exit 1
fi

if [[ "$patch_password_res" != *"\"updated_at\":\"20"* ]]; then
  printf "FAIL: Should return a valid date in the updated_at field\n"
  exit 1
fi

printf "\nUpdated user's password successfully\n\n"

# ============================================
# FAIL PATCH /users/{id} with invalid password
# ============================================
fail_patch_password_cmd="curl $opts --request PATCH --data '{\"password\": \"${bad_password}\"}' $auth_header $server/users/${user_id}"
printf "Update with bad password: $fail_patch_password_cmd\n"

fail_patch_password_res=$(eval $fail_patch_password_cmd)
echo $fail_patch_password_res

if [[ "$fail_patch_password_res" != *"\"error\":\"validation error: password\""* ]]; then
  printf "FAIL: Should return a validation error for password field\n"
  exit 1
fi

# ============================================
# FAIL GET /secret with invalid token
# ============================================
bad_auth_header="--header 'Authorization: Bearer xxx'"

fail_get_secret_cmd="curl $opts --request GET $bad_auth_header $server/secret"
printf "Get secret with bad token: $fail_get_secret_cmd\n"

fail_get_secret_res=$(eval $fail_get_secret_cmd)
echo $fail_get_secret_res

if [[ "$fail_get_secret_res" != *"\"error\":\"token invalid\""* ]]; then
  printf "FAIL: Should return a token invalid error\n"
  exit 1
fi

printf "\nALL TESTS PASSED!\n"
