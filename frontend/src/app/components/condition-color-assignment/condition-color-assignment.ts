import {Component, Input} from '@angular/core';
import {FormBuilder, FormGroup, ReactiveFormsModule} from "@angular/forms";

@Component({
  selector: 'app-condition-color-assignment',
  imports: [ReactiveFormsModule],
  templateUrl: './condition-color-assignment.html',
  styleUrl: './condition-color-assignment.scss',
})
export class ConditionColorAssignment {
  @Input() conditions: string[] = [];
  _form!: FormGroup;

  @Input() set form(value: FormGroup) {
    this._form = value;
  }

  get form(): FormGroup {
    return this._form;
  }

  constructor(private fb: FormBuilder) {
    this._form = this.fb.group({});
  }
}
