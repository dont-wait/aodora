export default function FieldErr({ message }: { message?: string }) {
  return message ? <p role="alert" className="text-error text-sm mt-1">{message}</p> : null
}
