import { h } from 'preact';

export default function Form({ _errors, _formData, items }: any) {
  const nameErr = _errors?.find((e: any) => e.field === 'name');
  const emailErr = _errors?.find((e: any) => e.field === 'email');
  return (
    <div class="form-page">
      <form>
        <input name="name" value={_formData?.name || ''} />
        {nameErr && <span class="field-error">{nameErr.message}</span>}
        <input name="email" value={_formData?.email || ''} />
        {emailErr && <span class="field-error">{emailErr.message}</span>}
        <button type="submit">Submit</button>
      </form>
      {items && <ul>{items.map((i: any) => <li key={i}>{i}</li>)}</ul>}
    </div>
  );
}
