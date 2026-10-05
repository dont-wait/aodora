import { Fragment } from 'react'

/** Hiển thị chuỗi có đánh dấu **đậm** đơn giản mà không cần dangerouslySetInnerHTML. */
export default function RichText({ text }: { text: string }) {
  return <>{text.split('**').map((part, i) => (i % 2 === 1 ? <strong key={i}>{part}</strong> : <Fragment key={i}>{part}</Fragment>))}</>
}
