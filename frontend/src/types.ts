export type User = {
  id: string
  name: string
  email: string
}

export type Session = {
  token: string
  user: User
}

export type PollOption = {
  id: string
  label: string
  votes: number
}

export type Poll = {
  id: string
  slug: string
  question: string
  options: PollOption[]
  totalVotes: number
  createdAt: string
  closesAt: string | null
  isClosed: boolean
  isOwner: boolean
}

export type AuthPayload = {
  name?: string
  email: string
  password: string
}

export type CreatePollPayload = {
  question: string
  options: string[]
  closesAt?: string
}
