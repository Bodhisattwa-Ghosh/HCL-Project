# GUVI submission checklist

The implementation is only one part of the brief. Before submitting, complete every item below with your own accounts and confirm each link works in an incognito browser.

## 1. Publish the repository

- Initialize Git if necessary: `git init`.
- Set your own repository URL/module name if you want to replace the placeholder module path in `backend/go.mod`.
- Commit the source, `README.md`, Docker files, and lockfiles - never `.env` files or secrets.
- Create a **public** GitHub repository and push it.
- Open the public repository in an incognito browser to make sure it is reachable.

## 2. Deploy the live app

- Provision a public frontend, Go backend, MongoDB, and Redis instance.
- Configure the production variables documented in `README.md`; use a new random `JWT_SECRET` and `COOKIE_SECURE=true`.
- Ensure `FRONTEND_ORIGIN` exactly matches the browser origin (including `https://`, no stray slash).
- Verify the public URL using two browsers/devices: create, share, vote, see results update live, reject a duplicate vote, and close the poll.
- Copy the final public application URL.

## 3. Record the mandatory 3-5 minute video

Use an unlisted YouTube video or a public Google Drive link. Include:

1. The full product flow: sign up/log in, create a poll, share it, vote from another browser, and show the live result update.
2. The challenge that gave you the most trouble and how you solved it. A strong honest example here is keeping Redis live counters recoverable from MongoDB while preventing duplicate votes.
3. Whether you used AI tools. Be precise: for example, use of an AI coding assistant for scaffolding, implementation suggestions, debugging, and review. Explain what you verified yourself.

Do not skip the video: the assignment explicitly marks it as mandatory. Make sure you understand the architecture and can explain it in technical interviews rather than reading from generated code.

## 4. Send the submission

Email the following to `devhiring@hclguvi.com`:

1. Public GitHub repository link.
2. Public live deployment link.
3. Public/unlisted 3-5 minute video link.

Suggested subject: `GUVI Developer Internship - <Your Name>`.

Suggested body:

```text
Hi GUVI team,

Here are my Developer Internship task submission links:

GitHub: <public repository URL>
Live app: <public deployment URL>
Video: <public or unlisted video URL>

Thank you,
<Your name>
```

Only send the email after testing all three links while signed out/incognito.
